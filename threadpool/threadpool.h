#ifndef __THREADPOOL__
#define __THREADPOOL__
#include <pthread.h>

typedef void* (*thread_func)(void *);

// 用于单元测试
//-----------------------------------------------------------
void test1(); // 测试task
void test_task_queue();
void test_taskget_more();
void test_taskget_create_more();  // 测试建立和销毁
void test_admin();  // 
//-----------------------------------------------------------

struct Task {
    thread_func task_func;
    void* args;
    struct Task* next;
};

struct thread_pool {
    // 任务队列
    int task_queue_size;
    int max_task_queue_size;
    struct Task* task_queue;
    int task_queue_head;
    int task_queue_tail;

    pthread_mutex_t mtx_pool;   // 只要对线程池做修改都要互斥
    pthread_cond_t task_queue_not_empty;  // 告诉线程池中阻塞的线程来任务了
    pthread_cond_t task_queue_not_full; // 告诉调用push_task的线程不用阻塞了，可以放任务了


    // 线程池
    int max_thread_pool_size;
    int min_thread_pool_size;
    int live_num;               // 已创建线程数量
    int busy_num;               // 已经忙碌于任务的线程数量


    pthread_t* thread_ids;      // 用于存放线程池中线程的ID；
    pthread_t admin;            // 监控线程的ID

    int shutdown;

    pthread_cond_t need_to_exit; // 告诉线程需要退出
    int need_to_exit_count;
};

// 初始化线程池
struct thread_pool* thread_pool_init(int, int, int);

// 任务队列的操作
void push_task(struct thread_pool*, thread_func, void*);
struct Task pop_task(struct thread_pool*);
void destroy_pool( struct thread_pool* ); //销毁pool，用大量的销毁测试程序是否崩溃
#endif