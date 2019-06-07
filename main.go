package main

import (
	"fmt"
	"os"
	"os/exec"
	"syscall"
)

func main() {

	if len(os.Args) != 3 {
		fmt.Println("Usage: go run main.go run /bin/sh")
		os.Exit(1)
	}

	flag := os.Args[1]
	command := os.Args[2]

	if flag == "run" {
		Run(command)
	}

	// run 命令内部调用 init
	if flag == "init" {
		Init(command)
	}
}

// 进程自己调用自己来创建隔离的进程
// 1. 创建/proc/self/exe init command 命令
// 2. 指定隔离的namespace，设置uid和gid
// 3. 分配伪终端
func Run(command string) {
	cmd := exec.Command("/proc/self/exe", "init", command)
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
	// 输入输出
	cmd.Stdin = os.Stdin
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	if err := cmd.Start(); err != nil {
		panic(err)
	}
	cmd.Wait()
}

// 初始化容器
// 1. 以priviate的方式挂载/proc目录
// 2. 执行用户输入的命令
func Init(command string) {
	// priviate 方式挂载，不影响宿主机的挂载
	syscall.Mount("", "/", "", syscall.MS_PRIVATE|syscall.MS_REC, "")

	// 挂载/proc目录
	mountFlags := syscall.MS_NOEXEC | syscall.MS_NOSUID | syscall.MS_NODEV
	syscall.Mount("proc", "/proc", "proc", uintptr(mountFlags), "")

	// linux execve系统调用: 启动一个新程序，替换原有进程，所以被执行进程的PID不会改变。
	if err := syscall.Exec(command, []string{command}, os.Environ()); err != nil {
		panic(err)
	}
	cmd := exec.Command(command)
	if err := cmd.Start(); err != nil {
		panic(err)
	}
	cmd.Wait()
}
