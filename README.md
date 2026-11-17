# gores — Go 二进制符号还原分析工具

针对去符号(stripped)Go 恶意程序的应急分析工具。从二进制中残留的 Go 运行时
元数据(pclntab / moduledata / buildinfo)恢复:

- **函数名 + 源文件路径 + 行号**(解析 `runtime.pclntab` 的 functab / funcnametab / cutab / filetab / pctab)
- **编译器版本**(buildinfo 或 `runtime.buildVersion` 回退扫描)
- **第三方模块及版本**(go buildinfo 的 deps 列表,含 replace 信息)
- **字符串常量 → 引用它的函数**(反汇编 `.text`,x86 跟踪 RIP 相对寻址,ARM64 跟踪 ADRP+ADD 对)
- **包级统计**,便于快速定位业务代码 vs 标准库

## 支持范围

| 容器格式 | ELF / PE / Mach-O |
|---|---|
| 架构 | amd64、386、arm64(字符串 xref);其他架构函数表仍可解析 |
| Go 版本 | pclntab go1.16 / go1.18 / go1.20 三种格式(Go 1.16 – 1.27 实测) |

## 目录结构

```
gores/            Go 解析器 + CLI + HTTP API
  internal/gores/   解析库(file/pclntab/functab/buildinfo/strings/xref)
web/              Nuxt 3 分析界面(上传 → 概览/函数/字符串/模块 四个视图)
samples/          自编译的去符号测试样本(8 个,含 go1.19/go1.27 两种 pclntab 格式)
```

## 使用

### CLI

```bash
cd gores && go build -o gores .
./gores analyze /path/to/sample.exe -o report.json
./gores analyze /path/to/sample.exe --line-tables   # 附带每函数完整行号表
```

### Web 界面

```bash
# 终端 1:后端(同时托管前端静态文件)
cd gores && ./gores serve -addr :8080 -web ../web/.output/public

# 前端开发模式(可选,/api 已代理到 8080)
cd web && npm install && npm run dev

# 前端生产构建
cd web && npm run generate   # 产物在 web/.output/public
```

浏览器打开 http://localhost:8080 ,拖入样本即可。

### 重建样本 / 跑测试

```bash
./samples/build.sh          # 需要本机 Go;go1.19 样本需 golang.org/dl/go1.19.13
cd gores && go test ./internal/gores/   # 对 samples/bin 全部样本做回归
```

## 实现要点 / 已知限制

- **字符串边界**:Go 把字符串字面量无分隔地拼接在 rodata 里,本工具以"被指令
  引用的地址"作为切分点还原单个字符串;未被引用的串可能与相邻串粘连显示。
- **go1.15 及更早**的 pclntab(go12 magic)暂未支持函数表解析(需要额外解析
  moduledata),会给出明确报错。近年样本基本都是 go1.18+。
- **32 位 ARM / PPC64** 的字符串 xref 未实现(函数表/构建信息不受影响)。
- 加壳(UPX 等)样本需先脱壳,pclntab 被压缩后无法直接定位。
