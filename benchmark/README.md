# testing

Script to compare go-extract and go-slug.

## Prepare environment

Setup local testing environment.

### Install graphiz for visualization

```shell
brew install graphviz
```

### Prepare test archives

The Terraform module [terraform-aws-modules/iam/aws](https://registry.terraform.io/modules/terraform-aws-modules/iam/aws/latest) from [Github](https://github.com/terraform-aws-modules/terraform-aws-iam/archive/refs/tags/v5.34.0.tar.gz) is used for testing.

The module is taken as-is from GitHub to provide a small archive as testing data. In addition, the archive is extended by 512Mb random bytes that are taken from `/dev/random`. The generated archives are `small-module.tar.gz` and `big-module.tar.gz`.

```shell
./prep.sh
```

## Perform tests and review graph in web

```shell
./prep.sh
mkdir runs
export CNT=100
go run main.go -v -p -P -e -s -i $CNT -o runs/$(date "+%Y-%m-%d_%H-%M-%S")_mem_$CNT.pprof *.gz
go tool pprof -http=:8080 runs/2024-02-06_09-18-46_mem_1000.pprof
```

## Helper script

A dedicated go script is created to compare `go-slug` and `go-extract`. The script can be used to generate a memory profile after execution.

```shell
Usage: main <input-archives> ... [flags]

Arguments:
  <input-archives> ...

Flags:
  -h, --help                       Show context-sensitive help.
  -c, --cache-in-memory
  -e, --extract
  -i, --iterations=1
  -p, --profile
  -o, --profile-out="mem.pprof"
  -P, --parallel
  -m, --src-from-mem
  -s, --slug
  -v, --verbose
```
