#ifndef __MYREDIS__ 
#define __MYREDIS__

#include <map>
#include <string>
using namespace std;
class myredis {
public:
    myredis() {
        mp.clear();
    }
    void put(string key, string value);
    string get(string key);
private:
    map<string, string> mp;
}; 

void myredis::put(string key, string value) {
    this->mp[key] = value;
}
string myredis::get(string key) { 
    if(this->mp.find(key) == this->mp.end()) {
        return "";
    }
    return this->mp[key];
}

#endif