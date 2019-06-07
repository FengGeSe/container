# container



## 1.0 namespace隔离

csdn: https://blog.csdn.net/qq_27068845/article/details/90708912 

六种namespace 分别是 :

- syscall.CLONE_NEWUTS 隔离主机名和域名
- syscall.CLONE_NEWIPC 隔离进程间通信
- syscall.CLONE_NEWPID 隔离进程ID
- syscall.CLONE_NEWNS 隔离挂载点和文件系统
- syscall.CLONE_NEWUSER 隔离用户的用户组ID
- syscall.CLONE_NEWNET 隔离网络设备



