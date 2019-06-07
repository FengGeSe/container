# container

本项目是在看了《自己动手写Docker》一书后，根据自己的理解。将实现容器引擎的关键代码按照自己的风格简单的写出来。为的是增强自己对docker实现的理解。仅供自己学习使用。

初期代码偏重实现容器功能，不考虑使用(比如做成命令行工具)。 后期深入了解docker源码后，再做封装。



### 环境准备

ubuntu: 16.04 LTS

golang: 1.12.5

在mac上使用vagrant+virtualbox创建的ubuntu镜像。 因为linux和mac的系统调用有点不同，这里直接在ubuntu内部使用vim写。后续上传 ubuntu+docker+go+vim-go 的镜像。



### 版本

#### 1.0 namespace隔离

csdn: [【实现简单的容器】- goalng实现namespace隔离的容器](https://blog.csdn.net/qq_27068845/article/details/90708912 )

```
git checkout 1.0_namespace
```

已实现六种namespace :

- syscall.CLONE_NEWUTS 隔离主机名和域名
- syscall.CLONE_NEWIPC 隔离进程间通信
- syscall.CLONE_NEWPID 隔离进程ID
- syscall.CLONE_NEWNS 隔离挂载点和文件系统
- syscall.CLONE_NEWUSER 隔离用户的用户组ID
- syscall.CLONE_NEWNET 隔离网络设备



#### 2.0 cgroup资源限制

csdn :[【实现简单的容器】- namespace隔离和cgroup资源限制](https://blog.csdn.net/qq_27068845/article/details/91043036)

```
git checkout 2.0_cgroup
```

已实现的资源限制：

- memory.limit_in_bytes  内存使用限制
- cpu.shares  CPU时间片权重



