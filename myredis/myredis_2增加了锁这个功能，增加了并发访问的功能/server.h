#ifndef __SERVER__
#define __SERVER__
#include <vector>
#include <set>
#include <poll.h>
#include <netinet/in.h> // for sockaddr_in
#include <sys/socket.h> // for socket()
#include <arpa/inet.h> // for htons()...
#include <sys/types.h> 
#include <cstring>
#include <string>
#include <unistd.h>
#include <sstream>
#include <stdio.h>
#include "myredis.h"
using  namespace std;

const int PUT = 0;
const int GET = 1;
class server;
struct opt {
    int op; 
    string key;
    string value;
    string info; // 如果op为-1说明出错，则要在这写为什么错
    opt(int op = 0, string key = "", string value = "", string info = "") 
    : op(op), key(key), value(value), info(info) {}
};

// 多线程传入的参数
struct opt_fd {
    string op;
    int fd;
    server *s;
    opt_fd(string op="",int fd=-1, server *s=NULL)
    : op(op),fd(fd), s(s){}
};

class server {
public:
    // 传入port列表，表示该服务器使用哪几个端口
    server(myredis * db,vector<int> & ports) 
    : db(db),nports((int)ports.size()), ports(ports)
    {
        for(int i = 0; i < ports.size(); i++) {
            this->pfd[i].fd = create_listen_fd(ports[i]);
            this->pfd[i].events = POLLIN;
        }
        for(int i = nports; i < this->maxsz; i++) {
            this->pfd[i].fd = -1;
        }
        
    }
    void run();
    
    string get( string key ) const {return this->db->get(key);}
    void put(string key, string value) const {this->db->put(key,value);} 

private:
    int create_listen_fd(int port); // 返回监听套接字
    static const int maxsz = 1000; // 最多使用多少个套接字
    pollfd pfd[maxsz];    // 需要监听的套接字序列
    int nports;          // 端口数
    set<int> con_set; // 管理已连接套接字在pfd中的坐标
    myredis* db; // 数据库
    vector<int> ports;
    char buf[1000]; // 暂存命令字符串
    static const int max_buf = 10000;
};
struct opt parse(string s) {
    //cout << "parse: " << s << endl;
    stringstream ss;
    ss << s;
    string field;
    opt ret;
    vector<string> v;
    char r[1000];
    while(ss >> field) {
        v.push_back(field);
    }
    bool ok = true;
    if (v.size() == 3 ) {
        if(v[0] == "PUT" || v[0] == "put") {
            ret.info = "OK";
            ret.op = PUT;
            ret.key = v[1];
            ret.value = v[2];
            sprintf(r, "[put] key=%s value=%s is ok!\n", ret.key.c_str(), ret.value.c_str());
            ret.info = string(r);
        } else ok = false;
    } else if(v.size() == 2 ) {
        if(v[0] == "GET" || v[0] == "get") {
            ret.info = "OK";
            ret.op = GET;
            ret.key = v[1];
            sprintf(r, "[get] key=%s\n", ret.key.c_str());
            ret.info = r;
        } else ok = false;
    } else {
        ok =false;
    }
    if (!ok) {
        ret.op = -1;
        ret.info = "You's Command is wrong, Don't Touch me!\n";
    }
    return ret;
}
// 这个函数好像不能作为成员函数，因为作为成员函数无法传入多线程那个函数里
// 多线程处理对数据的操作
static pthread_mutex_t mtx = PTHREAD_MUTEX_INITIALIZER;
void* deal(void *args) {
    opt_fd optfd = *((struct opt_fd *) args);
    opt _opt = parse(optfd.op);
    
    server *s = optfd.s;
    if(_opt.op == GET) {
        pthread_mutex_lock(&mtx);
        string res = s->get(_opt.key);
        pthread_mutex_unlock(&mtx);
        res += '\n';
        write(optfd.fd,res.c_str(),res.length());
        
    } else if (_opt.op == PUT){
        pthread_mutex_lock(&mtx);
        s->put(_opt.key,_opt.value);
        pthread_mutex_unlock(&mtx);
        write(optfd.fd,_opt.info.c_str(),strlen(_opt.info.c_str()));
    } else {
        write(optfd.fd,_opt.info.c_str(),strlen(_opt.info.c_str()));
        return (void*)-1;
    }
    // 防止内存泄漏
    delete (struct opt_fd *)args;
}






void server::run() {
    for( ; ; ) {
        //阻塞直到有一个可读
        if (poll(pfd,maxsz,-1)< 0) {  
            printf("err for select\n");
        }
        for(int p = 0; p < nports; p++) {
            if (this->pfd[p].revents == POLLIN) {
                int fd = pfd[p].fd;
                struct sockaddr_in cliaddr;
                socklen_t len = sizeof(cliaddr);
                int connfd =accept(fd, (struct sockaddr*)&cliaddr, &len);
                printf("通过%d端口连接到%s\n", this->ports[p],inet_ntoa(cliaddr.sin_addr));
                for(int i = nports; i < maxsz; i++) {
                    if(this->pfd[i].fd<0){
                        pfd[i].fd=connfd;
                        pfd[i].events=POLLIN;
                        this->con_set.insert(i);
                        break;
                    }
                }
            }
        }

        // 查看是否有写入的操作

        for(set<int>::iterator it = this->con_set.begin(); it != this->con_set.end(); ) {
            bool need_to_delete = false;
            int idx = *it;
            if(this->pfd[idx].revents == POLLIN) {
                int fd = this->pfd[idx].fd;
                int n;
                if((n = read(fd, this->buf, this->max_buf)) < 0) {
                    printf("err");
                } else if (n == 0) {
                    close(fd); //关闭该套接字
                    this->pfd[idx].fd = -1;
                    need_to_delete = true;
                    printf("已经关闭连接-%d\n",fd);
                } else {
                    this->buf[n-2] = '\0';
                }
                // 另外开一个线程对数据库做操作
                if(!need_to_delete) {
                    pthread_t tid;
                    string s = string(this->buf) ;
                    opt_fd *_opt_fd = new opt_fd(s,pfd[idx].fd, this);
                    pthread_create(&tid, NULL,deal, _opt_fd); // 创建线程执行操作
                }
            }
            // 删除已经断开的连接
            if(need_to_delete) {
                it = con_set.erase(it);
            } else {
                it++;
            }
        }


    }
}
int server::create_listen_fd(int port) {
    int listenfd, connfd;
    struct sockaddr_in servaddr, cliaddr;
    listenfd = socket(AF_INET, SOCK_STREAM, 0);
    memset(&servaddr, 0, sizeof(servaddr));
    servaddr.sin_family = AF_INET;
    servaddr.sin_addr.s_addr = htonl(0);
    servaddr.sin_port = htons(port);
    bind(listenfd, (struct sockaddr*) &servaddr, sizeof(servaddr));
    listen(listenfd,5);
    return listenfd;
}
#endif