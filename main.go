package main

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
)

var (
	groupName = "mydocker-limit"

	// docker run -d busybox top -b
	// docker export -o busybox.tar (容器ID)
	// tar -xvf busybox.tar -C /root/busybox
	imgPath = "/root/busybox"
)

func main() {

	if len(os.Args) < 2 {
		fmt.Println("Usage: go run main.go run /bin/sh")
		os.Exit(1)
	}

	flag := os.Args[1]

	if flag == "run" {
		command := os.Args[2]
		Run(command)
	}

	// run 命令内部调用 init
	if flag == "init" {
		Init()
	}
}

// 进程自己调用自己来创建隔离的进程
// 1. 创建/proc/self/exe init command 命令
// 2. 指定隔离的namespace，设置uid和gid
// 3. 分配伪终端
func Run(command string) {
	log.Printf("执行Run, 用户命令: %s", command)

	cmd := exec.Command("/proc/self/exe", "init")
	cmd.SysProcAttr = &syscall.SysProcAttr{
		// namespace
		Cloneflags: syscall.CLONE_NEWPID | syscall.CLONE_NEWUSER | syscall.CLONE_NEWUTS | syscall.CLONE_NEWIPC | syscall.CLONE_NEWNET | syscall.CLONE_NEWNS,
		// CLONE_NEWUSER 需要指定uid和gid
		UidMappings: []syscall.SysProcIDMap{
			{
				ContainerID: 0,
				HostID:      0,
				Size:        1,
			},
		},
		GidMappings: []syscall.SysProcIDMap{
			{
				ContainerID: 0,
				HostID:      0,
				Size:        1,
			},
		},
	}
	log.Printf("指定namespace的隔离")
	// 输入输出
	cmd.Stdin = os.Stdin
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout

	// 建立FIFO传递用户命令
	read, write, err := os.Pipe()
	if err != nil {
		panic(err)
	}
	write.WriteString(command)
	write.Close()
	cmd.ExtraFiles = []*os.File{read}
	log.Printf("建立FIFO传递用户命令")

	// 挂载rootfs
	cmd.Dir = imgPath

	if err := cmd.Start(); err != nil {
		panic(err)
	}
	log.Printf("开始调用init命令")

	// 设置cgroup, 限制内存限制和cpu
	memoryLimit := "50m"
	cpuShare := "512"
	SetCgroups(groupName, cmd.Process.Pid, memoryLimit, cpuShare)
	log.Printf("设置cgroup, memory limit: %s cpu share: %s\n", memoryLimit, cpuShare)
	defer RemoveCgroups(groupName)

	cmd.Wait()
}

// 初始化容器
// 1. 以priviate的方式挂载/proc目录
// 2. 执行用户输入的命令
func Init() {
	// 获得用户命令
	// uintptr(3)就是指index为3的文件描述符，也就是传递进来的管道的一端
	log.Printf("开始执行init命令")
	pipe := os.NewFile(uintptr(3), "pipe")
	msg, err := ioutil.ReadAll(pipe)
	if err != nil {
		panic(err)
	}
	command := string(msg)
	log.Printf("从管道中获取用户的命令: %s", command)

	// 切换root文件系统
	setUpMount()

	// priviate 方式挂载，不影响宿主机的挂载
	syscall.Mount("", "/", "", syscall.MS_PRIVATE|syscall.MS_REC, "")

	// 挂载/proc目录
	mountFlags := syscall.MS_NOEXEC | syscall.MS_NOSUID | syscall.MS_NODEV
	syscall.Mount("proc", "/proc", "proc", uintptr(mountFlags), "")

	// 调用exec.LookPath
	cmdArray := strings.Split(command, " ")
	path, err := exec.LookPath(cmdArray[0])
	if err != nil {
		panic(err)
	}

	// linux execve系统调用: 启动一个新程序，替换原有进程，所以被执行进程的PID不会改变。
	if err = syscall.Exec(path, cmdArray[0:], os.Environ()); err != nil {
		panic(err)
	}
	cmd := exec.Command(command)
	if err := cmd.Start(); err != nil {
		panic(err)
	}
	cmd.Wait()
}

func setUpMount() {
	// 当前路径
	pwd, err := os.Getwd()
	if err != nil {
		panic(err)
		return
	}

	err = pivotRoot(pwd)
	if err != nil {
		panic(err)
	}

	// mount proc
	defaultMountFlags := syscall.MS_NOEXEC | syscall.MS_NOSUID | syscall.MS_NODEV
	syscall.Mount("proc", "/proc", "proc", uintptr(defaultMountFlags), "")
	syscall.Mount("tmpfs", "/dev", "tmpfs", syscall.MS_NOSUID|syscall.MS_STRICTATIME, "mode=755")
}

// pivot_root把当前进程的root文件系统放到put_old目录，而使new_root成为新的root文件系统。
func pivotRoot(root string) error {
	if err := syscall.Mount(root, root, "bind", syscall.MS_BIND|syscall.MS_REC, ""); err != nil {
		return fmt.Errorf("Mount rootfs to itself error: %v", err)
	}
	// 创建rootfs/.pivot_root存储old_root
	pivotDir := filepath.Join(root, ".pivot_root")
	if err := os.Mkdir(pivotDir, 0777); err != nil {
		return err
	}
	// pivot_root到新的rootfs, 老的old_root现在挂载到rootfs/.pivot_root上
	// 挂载点现在依然能在mount命令中看到
	if err := syscall.PivotRoot(root, pivotDir); err != nil {
		return fmt.Errorf("pivot_root %v", err)
	}
	// 修改当前工作目录到根目录
	if err := os.Chdir("/"); err != nil {
		return fmt.Errorf("chdir error %v", err)
	}

	pivotDir = filepath.Join("/", ".pivot_root")
	// unmout rootfs/.pivot_root
	if err := syscall.Unmount(pivotDir, syscall.MNT_DETACH); err != nil {
		return fmt.Errorf("umount pivot_root dir error %v", err)
	}
	return os.Remove(pivotDir)
}

// @ pid  要被限制资源的pid
// @ memoryLimit memory.limit_in_bytes  内存限制
// @ cpuShare cpu.shares  CPU时间片权重
func SetCgroups(groupName string, pid int, memoryLimit, cpuShare string) {
	// memory.limit_in_bytes
	err := AddSubsystemLimit(pid, groupName, "memory", "memory.limit_in_bytes", memoryLimit)
	if err != nil {
		panic(err)
	}

	// cpu.shares
	err = AddSubsystemLimit(pid, groupName, "cpu", "cpu.shares", cpuShare)
	if err != nil {
		panic(err)
	}
}

// 移除cgroups
func RemoveCgroups(group string) {
	// 移除memory
	memoryCgroupPath, err := GetCgroupPath("memory", group, false)
	if err != nil {
		panic(err)
	}
	err = os.Remove(memoryCgroupPath)
	if err != nil {
		panic(err)
	}

	// 移除cpu
	cpuCgroupPath, err := GetCgroupPath("cpu", group, false)
	if err != nil {
		panic(err)
	}
	err = os.Remove(cpuCgroupPath)
	if err != nil {
		panic(err)
	}
}

func AddSubsystemLimit(pid int, group, subsystem, item, limit string) error {
	// 1. 获得cgroup的绝对路径，不存在则创建
	cgroupPath, err := GetCgroupPath(subsystem, group, true)
	if err != nil {
		return err
	}

	// 2. 写入限制
	err = ioutil.WriteFile(path.Join(cgroupPath, item), []byte(limit), 0644)
	if err != nil {
		return err
	}

	// 3. 将pid加入
	err = ioutil.WriteFile(path.Join(cgroupPath, "tasks"), []byte(strconv.Itoa(pid)), 0644)
	if err != nil {
		return err
	}
	return nil
}

// 通过/proc/self/mountinfo找出某个subsystem的hierarchy cgroup根节点所在的目录
func FindCgroupMountpoint(subsystem string) string {
	f, err := os.Open("/proc/self/mountinfo")
	if err != nil {
		return ""
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		txt := scanner.Text()
		fields := strings.Split(txt, " ")
		for _, opt := range strings.Split(fields[len(fields)-1], ",") {
			if opt == subsystem {
				return fields[4]
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return ""
	}
	return ""
}

// 得到cgroup在文件系统中的绝对路径
func GetCgroupPath(subsystem, cgroupPath string, autoCreate bool) (string, error) {
	cgroupRoot := FindCgroupMountpoint(subsystem)
	if cgroupRoot == "" {
		panic("不能是空")
	}
	if _, err := os.Stat(path.Join(cgroupRoot, cgroupPath)); err == nil || (autoCreate && os.IsNotExist(err)) {
		if os.IsNotExist(err) {
			if err := os.Mkdir(path.Join(cgroupRoot, cgroupPath), 0755); err == nil {
			} else {
				return "", fmt.Errorf("err create cgroup %v", err)
			}
		}
		return path.Join(cgroupRoot, cgroupPath), nil
	} else {
		return "", fmt.Errorf("cgroup path error %v", err)
	}
}
