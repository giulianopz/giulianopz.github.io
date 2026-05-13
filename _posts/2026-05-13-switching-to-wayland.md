---
layout: post
title:  "Reluctantly Switching to Wayland"
date:   2026-05-13
categories: wayland gnome x11
permalink: /reluctantly-switching-wayland
---

In preparation of upgrading to the current Ubuntu LTS (26.04), I'm reluctantly switching to [Wayland](https://wayland.freedesktop.org/). This will indeed be the only protocol supported (see [release notes](https://documentation.ubuntu.com/release-notes/26.04/summary-for-lts-users/#wayland-session)) from now on. [Mutter](https://mutter.gnome.org/) will be the Wayland [compositor](https://wiki.archlinux.org/title/Wayland#Compositors) on GNOME.

> Note: Compositors act as compositing window managers; they effectively replace what was called 'window managers' on Xorg. See [&#x00a7; X11 and Wayland](https://en.wikipedia.org/wiki/Compositing_manager#X11_and_Wayland) on Wikipedia for a brief historical overview.

Until today, I postponed this switch since not every piece of software my workflow depends on has done the necessary work to support it. I tend to be a zero-configuration person. But, unfortunately, a few tools in my toolbox turned out to be outdated and broken on Wayland.

The fact that this project drew the same amount of [hate](https://gist.github.com/probonopd/9feb7c20257af5dd915e3a9f2d1f2277) as other new components in the Linuxsphere (systemd, coff coff), did not encourage me to do it before.

This post is an attempt to document my journey to Wayland. It will come in handy since I have a number of Debian/Ubuntu machines to migrate similarly.

## Guake

First thing I noticed is that Guake can't be opened up anymore with its standard keyboard shortcut (`F12`). But it can be fixed by registering a new custom shortcut running the `toggle-guake` command.

That said, it seems to work decently except for not always opening in the last active display (see [issue#2115](https://github.com/Guake/guake/issues/2115)). Which is really annoying if you work with an external monitor...

I've considered switching to `ghostty`, but guess what? They are still arguing on who should implement what (see [this](https://github.com/ghostty-org/ghostty/discussions/3459#discussioncomment-13474811) and [this](https://gitlab.gnome.org/GNOME/mutter/-/work_items/973#note_668502)). Probably, I should just try to switch compositor (Sway, Hyprland, niri, river, Wayfire, ...) since they all seem to implement this [wlr layer shell](https://wayland.app/protocols/wlr-layer-shell-unstable-v1) interface.

> A little rant: in the past, Gnome developers also refused to add support for the Sixel image protocol to `gnome-terminal` (which is now deprecated - a twist of fate). Read [here](https://github.com/csdvrx/sixel-tmux/blob/main/RANTS.md#sixel-sabotage-in-vte). Why don't they want us to have some fun??

## rofi

There is a [fork](https://github.com/in0ni/rofi-wayland) for `rofi` on Wayland, but to make it work, a kludge can be enough: `WAYLAND_DISPLAY=`. It will force [X11/XWayland](https://wiki.archlinux.org/title/Wayland#Xwayland), which allows to run native X11 applications to run seamlessly in Wayland.


...