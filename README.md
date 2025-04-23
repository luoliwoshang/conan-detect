#### step
#### install rxsync
```bash
git clone https://github.com/goplus/rxsync
cd rxsync
go install ./...
```
#### clone conan center
```
git clone --depth 1 https://github.com/conan-io/conan-center-index.git
```
#### exec rxsync
`<conan-center-index>`: local path of conan-center-index
`<result-dir>`: rxsync result dir
```
rxsync init conan-ver-sync conanver:<conan-center-index> <result-dir>
rxsync run -walk conan-ver-sync
```
#### check github url


```bash
git clone https://github.com/luoliwoshang/conan-detect
cd conan-detect
go install ./...

check-conan-info list --dir=<result-dir> --all --count-github
```
result like this:
```bash
..............
=== Package 1835: zyre ===
Package: zyre
Version: 2.0.1
URLs: [https://github.com/zeromq/zyre/archive/v2.0.1.tar.gz]

=== Package 1836: zziplib ===
Package: zziplib
Version: 0.13.78
URLs: [https://github.com/gdraheim/zziplib/archive/refs/tags/v0.13.78.tar.gz]


=== Statistics ===
Packages with first URL starting with https://github.com/: 1442/1836
Packages with errors: 61/1836
```


----
method 1
```yml
sources:
  "1.3.1":
    url:
      - "https://zlib.net/fossils/zlib-1.3.1.tar.gz"
      - "https://github.com/madler/zlib/releases/download/v1.3.1/zlib-1.3.1.tar.gz"
```
method 2

```yml
sources:
  "1.2.11":
    url: "https://zlib.net/fossils/zlib-1.2.11.tar.gz"
    sha256: "c3e5e9fdd5004dcb542feda5ee4f0ff0744628baf8ed2dd5d66f8ca1197cb1a1"
```
