---
layout: post
title:  "Enforce Zero Allocations on Hot Paths in Go"
date:   2026-08-26
categories: go escape analysis checkescape allocation heap hot paths
permalink: /enforce-zero-alloc-go
---

A few days ago, I stumbled upon an interesting [thread](https://bsky.app/profile/eatonphil.bsky.social/post/3mpqwjdspek2c) on Bluesky started by [Phil Eaton](https://eatonphil.com/), and I found out something I didn't know about Go before. (That's good news in itself: please stop complaining about Bluesky/Mastodon being a poor replacement for Twitter.)

Phil shared a good [article](https://theconsensus.dev/p/2026/06/27/the-feature-in-oxcaml-more-languages-should-steal.html) that shows a nice feature of OxCaml:

> In most languages, you hunt down allocations with a profiler and they creep back the moment you touch the hot path.
> Jane Street's superset of OCaml lets you flip that around: annotate a function with `[@zero_alloc]` and the compiler refuses to build if anything in its call tree touches the heap.

He concludes the article by saying:

> This ability to mark call trees as not allocating is incredibly useful. It’s a great feature the Jane Street team added and I’d love to see more languages build this into their compilers.

It turns out that Go has something similar. Although not baked into the compiler.

When standard Go escape analysis runs, `go build` reports whether variables escape to the heap if you pass `-gcflags="-m"` (see [Go Wiki: Compiler And Runtime Optimizations](https://go.dev/wiki/CompilerOptimizations#escape-analysis-and-inlining)). This is already informative per se (and major IDEs, like [GoLand](https://blog.jetbrains.com/go/2026/07/20/escape-analysis/#running-the-escape-analysis-tool) and [VSCode](https://github.com/golang/vscode-go/wiki/settings#uidiagnosticannotations), make it available for you), but I didn’t know there was a way to control heap escapes, as well.

In the thread, [@thepudds](https://github.com/thepudds) (PJ Malloy, a well-known Go contributor) pointed out that the `testing` package has a function, [AllocsPerRun](https://pkg.go.dev/testing#AllocsPerRun), to explicitly count allocations during a benchmark run.

Being able to assert allocations at test-time could already be good enough (see [Ensuring zero allocations in Go tests](https://web.archive.org/web/20251002093518/https://justin.azoff.dev/blog/ensuring-zero-allocations-in-go-tests/)), but there’s something even better.

There is also a tool developed for the [gVisor](https://github.com/google/gvisor) project, named `checkescape`, that is similar in spirit to what OxCaml offers. It lets you annotate a function so that static analysis can detect whether it allocates on the heap.

To quote the [docs](https://pkg.go.dev/gvisor.dev/gvisor/tools/checkescape):

> Package checkescape allows recursive escape analysis for hot paths.
> The analysis tracks multiple types of escapes, in two categories. First, 'hard' escapes are explicit allocations. Second, 'soft' escapes are interface dispatches or dynamic function dispatches; these don't necessarily escape but they *may* escape.

`checkescape` also supports several options to enforce specific escape expectations (escape locality, escape type, escape requirement, and exemptions).

Unfortunately, it is tightly coupled to the project (bazel-based). But since `checkescape` is built on `golang.org/x/tools/go/analysis`, we can wrap it into a single checker to analyse a Go module directly as a standalone tool, or also with `go vet`. I sketched a wrapper for `checkescape` that you can find in this [PR](https://github.com/google/gvisor/pull/14181). This should make it easier to install:
```bash
go install gvisor.dev/gvisor/tools/checkescape/cmd/checkescape@go
```

The PR is not yet merged, so you have to check out its branch and build it yourself from source (`go build -o checkescape main.go`) for the moment.

To tell the analyzer to check a hot path, add `// +checkescape` directly above the target function:
```go
1 package main
2
3 import "fmt"
4
5 type Data struct {
6 	x int
7 	y int
8 	z int
9 }
10
11 var sink *Data
12
13 // +checkescape
14 func HeapAllocEscape() {
15 	d := &Data{1, 2, 3}
16 	sink = d // d escapes to heap
17 }
18
19 func main() {
20 	HeapAllocEscape()
21 	fmt.Println(sink)
22 }
```

Then run `checkescape`:
```bash
# as a standalone tool
$ checkescape ./...
# or as a vet tool
$ go vet -vettool=$(which checkescape) ./...
# output will be
main.go:15:12: heap: explicit allocation → runtime.newobject
```

If we print the disassembly of the code (filtering by the snippet start and end offsets), we can see the `CALL` instruction that `checkescape` interprets as an escape:
```bash
$ go tool objdump -S test.bin 0x49e140 0x49e1e5
TEXT main.main(SB) /tmp/checkescape-playground/main.go
func main() {
  0x49e140              493b6610                CMPQ SP, 0x10(R14)
  0x49e144              0f8693000000            JBE 0x49e1dd
  0x49e14a              55                      PUSHQ BP
  0x49e14b              4889e5                  MOVQ SP, BP
  0x49e14e              4883ec38                SUBQ $0x38, SP
        HeapAllocEscape()
  0x49e152              90                      NOPL
        d := &Data{1, 2, 3}
  0x49e153              488d05068a0100          LEAQ 0x18a06(IP), AX
  0x49e15a              e801f6f7ff              CALL runtime.newobject(SB)  # this is effectively a heap escape
  0x49e15f              48c70001000000          MOVQ $0x1, 0(AX)
  0x49e166              48c7400802000000        MOVQ $0x2, 0x8(AX)
  0x49e16e              48c7401003000000        MOVQ $0x3, 0x10(AX)
        sink = d // d escapes to heap
  0x49e176              833d239e100000          CMPL runtime.writeBarrier(SB), $0x0
  0x49e17d              7413                    JE 0x49e192
  0x49e17f              488b155a960e00          MOVQ 0xe965a(IP), DX
  0x49e186              e8b510feff              CALL runtime.gcWriteBarrier2(SB)
  0x49e18b              498903                  MOVQ AX, 0(R11)
  0x49e18e              49895308                MOVQ DX, 0x8(R11)
  0x49e192              48890547960e00          MOVQ AX, 0xe9647(IP)
        fmt.Println(sink)
  0x49e199              440f117c2428            MOVUPS X15, 0x28(SP)
  0x49e19f              488b153a960e00          MOVQ 0xe963a(IP), DX
  0x49e1a6              4c8d05737b0000          LEAQ 0x7b73(IP), R8
  0x49e1ad              4c89442428              MOVQ R8, 0x28(SP)
  0x49e1b2              4889542430              MOVQ DX, 0x30(SP)
        return Fprintln(os.Stdout, a...)
  0x49e1b7              488b1d32960e00          MOVQ os.Stdout(SB), BX
  0x49e1be              488d05a3170300          LEAQ go:itab.*os.File,io.Writer(SB), AX
  0x49e1c5              488d4c2428              LEAQ 0x28(SP), CX
  0x49e1ca              bf01000000              MOVL $0x1, DI
  0x49e1cf              4889fe                  MOVQ DI, SI
  0x49e1d2              e889b0ffff              CALL fmt.Fprintln(SB)
}
  0x49e1d7              4883c438                ADDQ $0x38, SP
  0x49e1db              5d                      POPQ BP
  0x49e1dc              c3                      RET
func main() {
  0x49e1dd              0f1f00                  NOPL 0(AX)
  0x49e1e0              e8dbf6fdff              CALL runtime.morestack_noctxt.abi0(SB)
```

The tool itself is a bit slow due to the reverse-engineering involved in the analysis process, which can only occur after compilation and disassembly are complete. These are the main steps:
1. It walks the [SSA](https://en.wikipedia.org/wiki/Static_single-assignment_form) built by `golang.org/x/tools/go/ssa` and classifies each instruction it recognizes into an escape category (see [checkescape.go:713](https://github.com/google/gvisor/blob/6467832316026ba33bf3989639ef2174565234be/tools/checkescape/checkescape.go#L713)): a direct heap allocation ([ssa.Alloc](https://pkg.go.dev/golang.org/x/tools/go/ssa#Alloc)), a built-in allocation function (e.g. `makemap`, `newobject`, etc.), a stack split, or a interface/[dynamic dispatch](https://en.wikipedia.org/wiki/Dynamic_dispatch).
2. Then it checks if there is really a [CALL](https://www.felixcloutier.com/x86/call) instruction (to `runtime.newobject/mallocgc/etc.`) at that given line for *some* of the candidates from the previous step (*some others* unconditionally escape). If the compiler proved the value stays on the stack, there's no `CALL`, and the candidate site is dropped as a false positive.
3. Finally, escapes are then propagated recursively across the whole call graph, so a hot-path function is flagged if any callee transitively escapes.

So, a performance penalty is paid for treating the compiled machine code as ground truth rather than reimplementing the compiler's internals. One can argue that this is more robust than parsing English diagnostic plain text, which has no guaranteed stability across Go releases. The latter is the approach taken by [jordanlewis/gcassert](https://github.com/jordanlewis/gcassert/blob/master/gcassert.go).

I opened an [issue](https://github.com/golang/go/issues/81004) (more a clarification question, than a real proposal) asking if `checkescape` could eventually land in the compiler in the near future, but it is clear that this is not the case: it would be a hard commitment for the Go team, leaking compiler internals and [magic comments](https://github.com/golang/go/issues/12312#issuecomment-137192328) into the language specification.

Actually, the possibility of making an otherwise valid program fail to compile due to unwanted allocations is a language design decision that simply does not abide in Go. After all, Go is not Zig.

[Ian Lance Taylor](https://www.airs.com/ian/) even pointed out that it would be better to just "examine the compiler output generated by the `-m` option" (like `gcassert`). Honestly, I don’t know which approach (reading disassembly vs debug prints) is best. For sure, Ian does know better than I.

Whatever you may think about it, the good news is that the compiler gets better and better at each release (see [Go 1.27 will make some allocations cheaper](https://lemire.me/blog/2026/08/15/go-1-27-will-make-some-allocations-cheaper/)), leaving these optimizations necessary only for performance-critical programs like gVisor itself.

---

References:
- [Introduction to the Go compiler](https://github.com/golang/go/blob/master/src/cmd/compile/README.md)
- [Go Wiki: Compiler And Runtime Optimizations](https://go.dev/wiki/CompilerOptimizations#escape-analysis-and-inlining)
- [Go Optimization Guide](https://goperf.dev/01-common-patterns/stack-alloc)
- [Golang Internals Resources](https://github.com/emluque/golang-internals-resources)
- [x86 and amd64 instruction reference](https://www.felixcloutier.com/x86/)