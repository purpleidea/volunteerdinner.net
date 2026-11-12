HUGO ?= hugo
GOLANG ?= go

.PHONY: all build server

all: build

build:
	$(HUGO) --gc --minify --panicOnWarning --printI18nWarnings --printPathWarnings
	cd tools/poster && $(GOLANG) run . --input ../../public/poster/index.json --output ../../public/poster.pdf --preview ../../public/poster.svg

server:
	$(HUGO) server --bind 127.0.0.1 --buildDrafts --disableFastRender
