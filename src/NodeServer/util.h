#ifndef _UTIL_H
#define _UTIL_H

#include "util/tc_common.h"

#define __FILENAME__ (strrchr(__FILE__, FILE_SEP[0]) ? (strrchr(__FILE__, FILE_SEP[0]) + 1):__FILE__)

#define FILE_FUN   __FILENAME__<<":"<<__FUNCTION__<<":"<<__LINE__<<"|"
#define FILE_FUN_STR  TC_Common::tostr(__FILENAME__)+":"+TC_Common::tostr(__FUNCTION__)+":"+TC_Common::tostr(__LINE__)+"|"

constexpr char SERVERTYPE_TARS_CPP[] = "tars_cpp";
constexpr char SERVERTYPE_TARS_NODE[] = "tars_node";
constexpr char SERVERTYPE_TARS_NODE_PKG[] = "tars_node_pkg";
constexpr char SERVERTYPE_TARS_JAVA_WAR[] = "tars_java_war";
constexpr char SERVERTYPE_TARS_JAVA_JAR[] = "tars_java_jar";
constexpr char SERVERTYPE_TARS_DNS[] = "tars_dns";
constexpr char SERVERTYPE_NOT_TARS[] = "not_tars";


/*
机器：
1）CPU使用率
	通过/proc/stat读取所有CPU的总user1、nice1、system1、idle1、iowait1、irq1、softirq1的值以及每个核的值	
	sleep 5秒
	再次读取/proc/stat得到所有CPU的user2、nice2、system2、idle2、iowait2、irq2、softirq2的值	
	used=(user2-user1)+(nice2-nice1)+(system2-system1)+(irq2-irq1)+(softirq2-softirq1)	
	total=used + (idle2-idle1)+(iowait2-iowait1)	
	计算CPU使用率=used*100/total	

	[running]mqq@10.135.13.100:/proc$ cat stat
		 user		nice system		idle		iowait  irq	 softirq stealstolen  guest			
	cpu  302076951 71674 75287771 33160062211 136608123 5984 14193833 0				0

2）cpu wa
	duse=(user2-user1)+(nice2-nice1)	
	dsys=(system2-system1)+(irq2-irq1)+(softirq2-softirq1)	
	didle=(idle2-idle1)	
	diow=(iowait2-iowait1)	
	Div=duse+dsys+idile+diow	
	divo2=Div/2	
	计算wa使用率=((100*diow)+divo2)/Div	

	[running]mqq@10.135.13.100:/$ vmstat
	procs -----------memory---------- ---swap-- -----io---- -system-- -----cpu------
	r  b   swpd   free   buff  cache   si   so    bi    bo   in   cs us sy id wa st
	0  0 130796 4882980 150388 1749696 0    0     2   117    0    0  1  0 98  0  0

3）CPU负载
	读取/proc/loadavg得到机器的1/5/15分钟平均负载，上报结果为5分钟平均负载乘以100

	[running]mqq@10.135.13.100:/proc$ cat loadavg 
	0.00 0.00 0.00 1/743 2139
	前三数字1、5、15分钟内平均进程数,第四值分子正运行进程数分母进程总数近运行进程ID号

4）内存使用率
	读取/proc/meminfo中MemTotal的值，单位为KB	
	内存使用率= 100*( f_mem_total - f_mem_free - f_buffer - f_cached + f_share_rss * 4 ) / f_mem_total

	f_share_rss的值需要按照下面的方式采集ipcs -mu |grep resident，取pages resident的值*4KB
	[running]mqq@10.135.13.100:/proc$ cat meminfo 
	MemTotal:        8052872 kB
	MemFree:         4871460 kB
	Buffers:          154544 kB
	Cached:          1671648 kB
	SwapCached:        12480 kB
	Active:          2299360 kB
	Inactive:         700816 kB
	Active(anon):    1122760 kB
	Inactive(anon):    51940 kB
	Active(file):    1176600 kB
	Inactive(file):   648876 kB
	Unevictable:           0 kB
	Mlocked:               0 kB
	SwapTotal:       2104504 kB
	SwapFree:        1973836 kB
	Dirty:               168 kB
	Writeback:             4 kB
	AnonPages:       1162020 kB
	Mapped:            23484 kB
	Shmem:               704 kB
	
进程：
1）CPU使用率
	通过/proc/<pid>/stat获取进程的pid，state，ppid，utime，stime，cutime，cstime，start_time，processor，
	并计算CPU使用率=((stat->utime + stat->stime) - (pre_stat->utime + pre_stat->stime))/采集间隔
	间隔 2秒

	[root@localhost ~]# cat /proc/6873/stat
	6873 (a.out) R 6723 6873 6723 34819 6873 8388608 77 0 0 0 41958 31 0 0 25 0 3 0 5882654 1409024 56 4294967295
	134512640 134513720 3215579040 0 2097798 0 0 0 0 0 0 0 17 0 0 0

	pid=6873 进程(包括轻量级进程，即线程)号
	comm=a.out 应用程序或命令的名字
	task_state=R 任务的状态，R:runnign, S:sleeping (TASK_INTERRUPTIBLE), D:disk sleep (TASK_UNINTERRUPTIBLE), T: stopped, T:tracing stop,Z:zombie, X:dead
	ppid=6723 父进程ID
	pgid=6873 线程组号
	sid=6723 c该任务所在的会话组ID
	tty_nr=34819(pts/3) 该任务的tty终端的设备号，INT（34817/256）=主设备号，（34817-主设备号）=次设备号
	tty_pgrp=6873 终端的进程组号，当前运行在该任务所在终端的前台任务(包括shell 应用程序)的PID。
	task->flags=8388608 进程标志位，查看该任务的特性www.linuxidc.com
	min_flt=77 该任务不需要从硬盘拷数据而发生的缺页（次缺页）的次数
	cmin_flt=0 累计的该任务的所有的waited-for进程曾经发生的次缺页的次数目
	maj_flt=0 该任务需要从硬盘拷数据而发生的缺页（主缺页）的次数
	cmaj_flt=0 累计的该任务的所有的waited-for进程曾经发生的主缺页的次数目
	utime=1587 该任务在用户态运行的时间，单位为jiffies
	stime=1 该任务在核心态运行的时间，单位为jiffies

2) 内存使用大小
	需要到/proc/进程id/statm
	内存使用大小=Resident*4KB

	[running]mqq@10.135.13.100:/proc/19905$ cat statm
	204058 160521 877 659 0 195342 0
	7组数字，每一个的单位都是一页 （常见的是4KB）分别是：
	size:任务虚拟地址空间大小
	Resident：正在使用的物理内存大小
	Shared：共享页数
	Trs：程序所拥有的可执行虚拟内存大小
	Lrs：被映像倒任务的虚拟内存空间的库的大小
	Drs：程序数据段和用户态的栈的大小
	dt：脏页数量

3）内存使用率
	进程使用的物理内存 / 机器总物理内存
*/

#endif
