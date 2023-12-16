---
layout: post
title:  "Go Data Structures"
date:   2023-12-16
categories: go data structures
permalink: /go-data-structures
comments_id: 3
---

Few days ago, I wrote something really stupid in a group chat on Telegram I am in with other Go persons. For the umpteenth time, I fell into the [trap](https://go-traps.appspot.com/#overview) of thinking that you can somehow (???) iterate over a map in sorted order. This is not possible, of course, but I realized it only after sending the message unfortunately... 

I like to blame Java for introducing such misconsceptions in my head, but I guess it's my fault. However that may be, I take it as an oppurtunity to make an inventory of the ideas I've accumulated without a proper organization, at least for what concerns data structures in Go.

---

References:
- [Go Slices: usage and internals](https://go.dev/blog/slices-intro)
- [Go maps in action](https://go.dev/blog/maps)
- [Go Wiki: SliceTricks](https://go.dev/wiki/SliceTricks)
- [How the Go runtime implements maps efficiently (without generics)](https://dave.cheney.net/2018/05/29/how-the-go-runtime-implements-maps-efficiently-without-generics)
