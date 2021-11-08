main: main.o threadpool.o
	gcc -g -lpthread main.o threadpool.o -o main

main.o: main.c threadpool.h
	gcc -g -c -lpthread main.c

threadpool.o: threadpool.c threadpool.h
	gcc -g -c -lpthread threadpool.c
