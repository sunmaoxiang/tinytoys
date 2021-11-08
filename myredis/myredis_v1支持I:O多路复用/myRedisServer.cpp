#include <iostream>
#include "myredis.h"
#include <vector>
#include "server.h"
using namespace std;


int main() {
    myredis *db = new myredis();
    // 初始化参数
    vector<int> ports {8080,8090};
    server *s = new server(db, ports);
    s->run();
}