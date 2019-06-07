# container

本项目是在看了《自己动手写Docker》后，根据自己的理解。将实现容器引擎的关键代码按照自己的风格简单的写出来。为的是增强自己对docker实现的理解。仅供自己学习使用。

初期代码偏重实现容器功能，不考虑使用(比如做成命令行工具)。 后期深入了解docker源码后，再做封装。

## 1.0 namespace隔离

csdn: https://blog.csdn.net/qq_27068845/article/details/90708912 

六种namespace 分别是 :

- syscall.CLONE_NEWUTS 隔离主机名和域名
- syscall.CLONE_NEWIPC 隔离进程间通信
- syscall.CLONE_NEWPID 隔离进程ID
- syscall.CLONE_NEWNS 隔离挂载点和文件系统
- syscall.CLONE_NEWUSER 隔离用户的用户组ID
- syscall.CLONE_NEWNET 隔离网络设备









