---
layout: post
title:  "Join Zoom Meeting from PWA with One Click"
date:   2025-08-27
last_revision: 2025-09-07
categories: zoom pwa gnome mime
permalink: /join-zoom-meeting-pwa-1-click
comments_id: 6
---

Have you ever used the Zoom desktop app on Linux? If not, just [google](https://www.google.com/search?q=Zoom+desktop+app+on+Linux+cpu+memory&udm=14) it: it's basically a DoS attack.

Zoom people do not like you not using their desktop app, so if you don't have it installed, you will be prompted to install it by the browser when you click on a link to join a meeting (on Slack, for example).

That's really annoying… Most of the time, I have to manually copy the link and paste it into the Zoom PWA on the browser.

It turns out that it's not so difficult to intercept the click on the Zoom link and feed to the browser a modified version of the URL that directly throws you into the meeting.

All you need to do is have a simple bash script:
```
#!/bin/bash

ACTION=$(echo $1 | grep -Po '(?<=zoom.us\/)(join|start)')
ROOM_ID=$(echo $1 | grep -Po '(?<=confno=)\d+')
PWD=$(echo $1 | grep -Po '(?<=pwd=)[\w\.]+')
/usr/bin/firefox "https://app.zoom.us/wc/${ROOM_ID}/${ACTION}?pwd=${PWD}&fromPWA=1" &>/dev/null
```

Then, you need to create a [desktop entry](https://wiki.archlinux.org/title/Desktop_entries) for it:
```
[Desktop Entry]
Version=1.0
Name=Join Zoom Meeting
Exec=/usr/local/bin/join-zoom-meeting.sh %u
Terminal=false
NoDisplay=true
Type=Application
MimeType=x-scheme-handler/zoommtg;
StartupNotify=true
Actions=new-window;new-private-window;
```

Give this file a name with a `.desktop` extension and put it into `$HOME/.local/share/applications/`.

It will instruct the desktop environment to open links with the MIME type `zoommtg` with the script declared in the `Exec` field.

What is left to do is just to associate them by default:
```
$ xdg-mime default .local/share/applications/join-zoom-meeting.desktop x-scheme-handler/zoommtg
# double-check
$ xdg-mime query default x-scheme-handler/zoommtg
.local/share/applications/join-zoom-meeting.desktop
# refresh the desktop cache
$ update-desktop-database $HOME/.local/share/applications/
```

And that's it! Next time you click `Join` on Slack, your browser will teleport you into the meeting with one click.