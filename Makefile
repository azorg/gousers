# File: "Makefile"

PRJ := gousers
BIN := bin
OUT := $(BIN)/$(PRJ)

INSTALL_PREFIX := /usr/local

GIT_MESSAGE = "auto commit"

.PHONY: all help run clean distclean rebuild install uninstall fmt commit tidy vendor

all: $(OUT)

rebuild: clean all

help:
	@echo "make all        - full build (by default)"
	@echo "make rebuild    - clean and full rebuild"
	@echo "make go.mod     - generate go.mod"
	@echo "make go.sum     - generate go.sum"
	@echo "make help       - help"
	@echo "make run OPT='' - run with set of command OPTions"
	@echo "make clean      - clean"
	@echo "make distclean  - full clean"
	@echo "make install    - install to $(PREFIX)/$(BIN)"
	@echo "make uninstall  - uninstall"
	@echo "make fmt        - format Go sources"
	@echo "make commit     - auto commit by git"
	@echo "make tidy       - automatic update go.sum by tidy"
	@echo "make vendor     - create vendor"

clean:
	rm -f $(OUT)

distclean: clean
	rm -rf $(BIN)
	rm -f go.mod
	rm -f go.sum
	@#sudo rm -rf go/pkg
	rm -rf vendor
	go clean -modcache

install: all
	mkdir -p $(INSTALL_PREFIX)/$(BIN)
	cp $(OUT) $(INSTALL_PREFIX)/$(BIN)/

uninstall:
	rm -f $(INSTALL_PREFIX)/$(BIN)/$(PRJ)

fmt:
	@#echo "*** Format Go sources ***"
	@go fmt cmd/gousers/*.go
	@go fmt pkg/utmp/*.go

commit:
	git add .
	git commit -am $(GIT_MESSAGE)
	git push

go.mod:
	@echo ">>> create go.mod"
	@go mod init $(PRJ)

go.sum: go.mod
	@echo ">>> create go.sum"
	@go get github.com/fsnotify/fsnotify # Go library to filesystem notifications 

tidy: go.mod
	@echo ">>> automatic update go.sum by tidy"
	@go mod tidy # automatic update go.sum

vendor: go.sum
	@echo ">>> create vendor"
	@go mod vendor

run: go.mod go.sum
	@echo ">>> run $(CMD) $(OPT)"
	@cd cmd/$(CMD) && go run . $(OPT)

$(OUT): go.mod go.sum cmd/gousers/*.go \
        pkg/utmp/*.go pkg/signal/*.go
	@echo ">>> build $(OUT)"
	@mkdir -p $(BIN)
	@go build -o $(BIN) $(PRJ)/cmd/$(PRJ)/

# EOF: "Makefile"
