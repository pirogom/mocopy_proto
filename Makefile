clean:
	go clean
32:
	GOARCH=386 go build -ldflags '-s -w -H windowsgui' -o MoCOPY_x86.exe
64:
	GOARCH=amd64 go build -ldflags '-s -w -H windowsgui' -o MoCOPY_x64.exe
rsrc:
	rsrc -manifest MoCOPY.manifest -ico icon.ico -o MoCOPY.syso
all:
	make 32
	make 64
