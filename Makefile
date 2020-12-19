CFLAGS = $(shell pkg-config --cflags --libs gtk+-3.0)

all: bin/light_controller

bin/light_controller: obj/light_controller.o
	mkdir -p bin
	gcc -o $@ $? $(CFLAGS)

obj/light_controller.o: light_controller.c
	mkdir -p obj
	gcc -c -o $@ $< $(CFLAGS)
	
