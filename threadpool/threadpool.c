#include "threadpool.h"
#include <stdio.h>
#include <unistd.h>
#include <stdlib.h>
#include <signal.h>
#include <errno.h>
//----------------------------------------------------------
void* p(void *a) {
    char *s = (char*)a;
    printf("%s\n", s);
    return (void*)0;
}
void test1() {
    struct Task task;
    task.task_func = p;
    pthread_t t;
    pthread_create(&t,NULL,task.task_func,"abc");
    pthread_join(t,NULL);
}
void test_task_queue() {
    struct thread_pool* tp;
    tp = thread_pool_init(10,10,10);
    push_task(tp, p, "smx");
    push_task(tp, p, "is shuaige");
    sleep(1);
    printf("queue size : %d\n", tp->task_queue_size);
}
// 测试添加任务后任务的运行情况
void test_taskget_more() {
    // ./main |  sort -n -k 3 -t ' ' 使用这个命令测试
    struct thread_pool* tp;
    tp = thread_pool_init(10,10,10);
    for(int i = 0; i < 10; i++) {
        char *s = (char*)malloc(50);
        sprintf(s, "this is %d", i);
        push_task(tp, p, s);
    }
    sleep(3);
    printf("busy: %d\n", tp->busy_num);  // err :busy: -1
    printf("queue size: %d\n", tp->task_queue_size);
    printf("queue size: %d\n", tp->live_num);
}
// ok
void test_taskget_create_more() {
    for(int i = 0; i < 100; i++) { 
        printf("---------------------------------%d-----------------------\n", i);
        struct thread_pool* tp;
        tp = thread_pool_init(10,10,15);
        for(int i = 0; i < 100; i++) {
            char *s = (char*)malloc(10);
            sprintf(s, "this is %d", i);
            push_task(tp, p, s);
        }
        sleep(4);

        printf("busy: %d\n", tp->busy_num);  // err :busy: -1
        printf("queue size: %d\n", tp->task_queue_size);
        printf("queue size: %d\n", tp->live_num);
        destroy_pool(tp);
    }
}
// test admin, have some bug!
void test_admin() {
    struct thread_pool* tp;
    tp = thread_pool_init(20, 10, 150);
    sleep(3);
    printf("----- now live thread: %d\n", tp->live_num);
    for(int i = 0; i < 50; i++) {
        char *s = (char*)malloc(10);
        sprintf(s, "this is %d", i);
        push_task(tp, p, s);
        printf("now live thread: %d\n", tp->live_num);
    }
    sleep(3);
    printf("now live thread: %d\n", tp->live_num);
    sleep(4);
    printf("now live thread: %d\n", tp->live_num);
    sleep(10);
    printf("busy : %d\n", tp->busy_num);
    printf("now live thread: %d\n", tp->live_num);


}
//----------------------------------------------------------
// 任务队列的操作
void push_task(struct thread_pool* tp, thread_func tf, void* args) {
    // 防止虚假唤醒
    while(tp->task_queue_size == tp->max_task_queue_size) {
        pthread_cond_wait(&tp->task_queue_not_full, &tp->mtx_pool);
    }
    tp->task_queue_size++;
    struct Task* tmp = &tp->task_queue[tp->task_queue_tail];
    tp->task_queue_tail = (tp->task_queue_tail + 1) % tp->max_task_queue_size; //  环形队列
    tmp->task_func = tf;
    tmp->args = args;
    pthread_cond_signal( &tp->task_queue_not_empty ); // 发出信号告诉阻塞线程来任务了，别睡了，赶紧干活
    pthread_mutex_unlock(&tp->mtx_pool);
}

struct Task pop_task(struct thread_pool* tp) {
    tp->task_queue_size--;
    struct Task* retp = &tp->task_queue[tp->task_queue_head++];
    tp->task_queue_head %= tp->max_task_queue_size;
    struct Task ret =*retp;
//    free(retp->args);

    if(tp->task_queue_size == tp->max_task_queue_size - 1) pthread_cond_signal(&tp->task_queue_not_full);
    return ret;
}

void* thread_run(void *args) {
    struct thread_pool* tp = (struct thread_pool*) args;
    tp->live_num++;
    for(;;) {
        while(tp->task_queue_size == 0 && tp->need_to_exit_count == 0) {
            pthread_cond_wait(&tp->task_queue_not_empty, &tp->mtx_pool);  // 获取后会阻塞mtx_task_queue防止又添加任务            
        }
        if(tp->need_to_exit_count > 0 && tp->live_num > tp->min_thread_pool_size) {
            tp->live_num--;
            tp->need_to_exit_count--;
            pthread_mutex_unlock(&tp->mtx_pool);
            pthread_exit(NULL);
        } else if(tp->need_to_exit_count > 0) {
            continue;
        }
        if(tp->shutdown) {
            break;
        }
        struct Task task = pop_task(tp);

        tp->busy_num++;
        pthread_mutex_unlock(&tp->mtx_pool);
        
        (*(task.task_func))(task.args);  //运行这个task
        
        pthread_mutex_lock(&tp->mtx_pool);
        tp->busy_num--;
        pthread_mutex_unlock(&tp->mtx_pool);
    }
    pthread_exit(NULL);
}

/*线程是否存活*/
int is_thread_alive(pthread_t tid) {
    int kill_rc = pthread_kill(tid, 0);     //发送0号信号，测试是否存活
    if (kill_rc == ESRCH) {
        return 0;
    }
    return 1;
}

#define SLEEP_TIME 3
void* admin_run(void *args) {
    struct thread_pool* tp = (struct thread_pool*)args;
    sleep(SLEEP_TIME);
    for(;!tp->shutdown ;){
        sleep(SLEEP_TIME);
        pthread_mutex_lock(&tp->mtx_pool);
        int busy_num = tp->busy_num;
        int live_num = tp->live_num;
        int max_thread_pool_size = tp->max_thread_pool_size;
        pthread_mutex_unlock(&tp->mtx_pool);
        // 如果都在忙，并且线程还没有开完
        if(busy_num == live_num && live_num < max_thread_pool_size) {
            int need_to_add = tp->task_queue_size - live_num;
            for(int i = 0; i < tp->max_task_queue_size && need_to_add; i++) {
                if(!is_thread_alive(tp->thread_ids[i])) {
                    pthread_create(&tp->thread_ids[i], NULL, thread_run, tp);
                   
                    need_to_add--;
                }
            }
        }
        //这部分没写好
        //如果有很多空闲的，关掉一部分
        else if(busy_num * 2 <= live_num)  {
            pthread_mutex_lock(&tp->mtx_pool);
            tp->need_to_exit_count = live_num * 0.25;  //关掉25%的线程
            pthread_mutex_unlock(&tp->mtx_pool);
            pthread_cond_signal(&tp->task_queue_not_empty);
        }
    }
    pthread_exit(NULL);
}

// 初始化线程池
struct thread_pool* thread_pool_init(int max_task_queue_size, int min_thread_pool_size, int  max_thread_pool_size) {
    struct thread_pool* tp = (struct thread_pool*)malloc(sizeof(struct thread_pool));
//    // 任务队列
//    int task_queue_size;
//    int max_task_queue_size;
//    struct Task* task_queue;
//    int task_queue_head;
//    int task_queue_tail;
//
//    pthread_mutex_t mtx_pool;   // 只要对线程池做修改都要互斥
//    pthread_cond_t task_queue_not_empty;  // 告诉线程池中阻塞的线程来任务了
//    pthread_cond_t task_queue_not_full; // 告诉调用push_task的线程不用阻塞了，可以放任务了
    tp->task_queue_size = 0;
    tp->max_task_queue_size = max_task_queue_size;
    tp->task_queue = (struct Task*)malloc(sizeof(struct Task) * max_task_queue_size );
    tp->task_queue_head = 0;
    tp->task_queue_tail = 0;
    pthread_mutex_init(&tp->mtx_pool, NULL);
    pthread_cond_init(&tp->task_queue_not_empty, NULL);
    pthread_cond_init(&tp->task_queue_not_full, NULL); // 不能使用broadcast，防止虚假唤醒
//    // 线程池
//    int max_thread_pool_size;
//    int min_thread_pool_size;
//    int live_num;               // 已创建线程数量
//    int busy_num;               // 已经忙碌于任务的线程数量
    tp->max_thread_pool_size = max_thread_pool_size;
    tp->min_thread_pool_size = min_thread_pool_size;
    tp->live_num = 0;
    tp->busy_num = 0;
//    pthread_t* thread_ids;      // 用于存放线程池中线程的ID；
//    pthread_t admin;            // 监控线程的ID
    tp->thread_ids = (pthread_t*)malloc( sizeof(pthread_t) * max_thread_pool_size  );
    for(int i = 0; i < min_thread_pool_size; i++) {
        pthread_create(&tp->thread_ids[i], NULL, thread_run, tp);
    }
    //pthread_create(&tp->admin, NULL, admin_run, tp);
    tp->shutdown = 0;
    pthread_cond_init(&tp->need_to_exit, NULL);
    tp->need_to_exit_count = 0;
    return tp;
}
void destroy_pool( struct thread_pool* tp) {
    free(tp->task_queue);
    free(tp->thread_ids);
    pthread_mutex_destroy(&tp->mtx_pool);
    pthread_cond_destroy(&tp->task_queue_not_empty);
    tp->shutdown = 1;
    
    for(int i = 0; i < tp->min_thread_pool_size; i++) {
        if(is_thread_alive(tp->thread_ids[i])) {
            pthread_kill(tp->thread_ids[i], 9);
        }
    }
    //pthread_join(tp->admin, NULL);
    free(tp);
}

