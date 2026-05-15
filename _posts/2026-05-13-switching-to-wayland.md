---
layout: post
title:  "Reluctantly Switching to Wayland"
date:   2026-05-13
last_modified_at: 2026-05-15
categories: wayland gnome x11
permalink: /reluctantly-switching-wayland
---

In preparation for upgrading to the current Ubuntu LTS (26.04), I'm reluctantly switching to [Wayland](https://wayland.freedesktop.org/). This will indeed be the only protocol supported (see [release notes](https://documentation.ubuntu.com/release-notes/26.04/summary-for-lts-users/#wayland-session)) from now on. [Mutter](https://mutter.gnome.org/) will be the Wayland [compositor](https://wiki.archlinux.org/title/Wayland#Compositors) on GNOME.

> Note: Compositors act as compositing window managers; they effectively replace what was called 'window managers' on Xorg. See [&#x00a7; X11 and Wayland](https://en.wikipedia.org/wiki/Compositing_manager#X11_and_Wayland) on Wikipedia for a brief historical overview.

Until today, I postponed this switch since a few tools my workflow depends on are based on X11, and the people who maintain it have done little or nothing to adapt to the new normal. Or, at least, this was the impression I got when I temporarily enabled Wayland on GNOME.

The fact that this project drew the same amount of [hate](https://gist.github.com/probonopd/9feb7c20257af5dd915e3a9f2d1f2277) as other new components in the Linux sphere (systemd, coff coff), did not encourage me to do it before... But on a closer inspection, this happens all the time, and, beyond that, it was [ineluctable](https://ajaxnwnk.blogspot.com/2020/10/on-abandoning-x-server.html) and even desirable.

This post is an attempt to document my journey to Wayland. It will come in handy since I have a number of Debian/Ubuntu machines to migrate similarly.

## Guake

The first thing I noticed is that Guake can't be opened up anymore with its standard keyboard shortcut (`F12`). But, luckily, this can be fixed by registering a new custom shortcut which runs the `toggle-guake` command.

That said, it works decently, except it does not always open in the last active display (see [issue#2115](https://github.com/Guake/guake/issues/2115)). Which is really annoying if you work with an external monitor...

I've considered switching to `ghostty`, but guess what? They are still arguing on who should implement what (see [this](https://github.com/ghostty-org/ghostty/discussions/3459#discussioncomment-13474811) and [this](https://gitlab.gnome.org/GNOME/mutter/-/work_items/973#note_668502)). Probably, I should just try to switch compositor (Sway, Hyprland, niri, river, Wayfire, ...) since they all seem to implement the [wlr layer shell](https://wayland.app/protocols/wlr-layer-shell-unstable-v1) protocol, which clients can use to *create surfaces that are layers of the desktop* (like panels, lock screens, wallpapers, on-screen keyboards, notifications, launchers, drop-down terminals etc.).

GNOME/Mutter developers have made it substantially clear their refusal to implement layer-shell protocol (see [issue#1141](https://gitlab.gnome.org/GNOME/gnome-shell/-/work_items/1141)). TL;DR: they want this kind of client to run as GNOME shell extensions.

> A little rant: in the past, Gnome developers also refused to add support for the Sixel image protocol (or any other image protocol) to `gnome-terminal` (which is now deprecated - a twist of fate). Read [here](https://github.com/csdvrx/sixel-tmux/blob/main/RANTS.md#sixel-sabotage-in-vte). Why don't they want us to have some fun??

Unfortunately, many new projects (and forks of old ones) are adopting it, and GNOME is one of the few desktop environments that does not implement it. That's a big problem considering the amount of Linux desktops running Debian/Ubuntu and their derivatives...

## rofi

`rofi` supports Wayland since last year (2025) thanks to [@lbonn](https://github.com/lbonn) (kudos!), but it needs the layer shell protocol... which GNOME lacks. Unsetting the `WAYLAND_DISPLAY` env variable will make it work, forcing [X11/XWayland](https://wiki.archlinux.org/title/Wayland#Xwayland), which allows running native X11 applications seamlessly in Wayland.

But this only works when `rofi` is run from a terminal emulator; it will fail to process any user input when executed from a keyboard shortcut (see [issue#2214](https://github.com/davatorium/rofi/issues/2214) and the related [discussion](https://github.com/davatorium/rofi/discussions/2216)).

Refusing to ditch `rofi` for any other launchers (or one of its forks), I dug into Wayland documentation, and I found out that there's actually a actionable second path for clients like `rofi` which cannot rely on the layer shell on GNOME: the [XDG shell](https://wayland.app/protocols/xdg-shell) protocol.

Patching the `rofi` code would have meant for me to gain a substantial understanding of the complex choreography involved in interacting with the Wayland API (read [here](https://bugaevc.gitbooks.io/writing-wayland-clients/content/beyond-the-black-square/xdg-shell.html)). So, I tried to instruct Claude to patch `rofi`, and I was surprised to see it succeeding after a few attempts: what a time to be alive. Ye, I say this with both enthusiasm and skepticism. Anyway, the patch is [here](https://github.com/giulianopz/rofi-xdg-shell/tree/2214) in case you want to give it a try (please, send feedback). I will consider sending it to the upstream when I'm sure I can understand better what it does. But I've been using it for 2/3 days, and I can say that it works, at least.


To be continued...

---

References:

- [Wayland Explorer](https://wayland.app/protocols/)
- [Writing Wayland clients, Sergey Bugaev](https://bugaevc.gitbooks.io/writing-wayland-clients/content/)
- [Can I finally start using Wayland in 2026?, Michael Stapelberg](https://michael.stapelberg.ch/posts/2026-01-04-wayland-sway-in-2026/)