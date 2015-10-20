#!/bin/sh

set -e

if [ -e go-moremath ]; then
    mv go-moremath go-moremath.old
fi

git clone --depth=1 http://github.com/aclements/go-moremath
rm -rf go-moremath/.git
sed -i -e 's,github.com/aclements/\(go-moremath\),rsc.io/benchstat/internal/\1,' $(find -name \*.go)
