---
layout: post
title:  "BIOS Password Recovery for Laptops"
date:   2024-11-19
last_revision: 2024-11-19
categories: acer aspire bios password
permalink: /bios-pwd-recovery-laptops
comments_id: 5
---

Hi, this is mostly self-referential.

I was upgrading the RAM (4GB->16GB) and the disk (HDD->SDD) of an old laptop of mine that is currently used by my parents. This laptop is an Acer Aspire E15 (ES1-571 series).

After installing Debian 12 on the new disk, I found myself stupefied again by the creepy message "No Bootable Device" when trying to reboot the machine.

This is at least the third time that I reinstall Linux on this laptop but I keep forgetting what is causing this mess. Hence, this brief post.

As it turns out, this series of Acer laptops requires you to select an UEFI file as trusted: you have to enter the BIOS menu and manually select the path of **shimx64.ef**i (e.g. `EFI\debian\shimx64.efi`), a shim that launches GRUB on computers which have Secure Boot active, in the Security panel.

The real problem is that you have to remember the "supervisor password" to do it...

I don't even think I've never set it... 

Many answers on StackOverflow tell you that there's absolutely no way to recover such password.

Fortunately, this is not true: the right answer was just some Google queries away.

I stumbled upon a YouTube video that reminded of the existence of this site: `https://bios-pw.org/#`

This is basically a collection of scripts that can generate the password you need if you pass to it the key that the BIOS will print after entering an invalid password for three times. More technical details in the article linked in the references.

And [that's that](https://www.youtube.com/watch?v=bX4Hml7aUvU)!

Thanks people of the web, as usual. 

Hopefully, this post can help other persons burden with the responsibility of being the sysadmins of their parents, just like me.

### a small side note

Anyway, the RAM and SSD upgrade brought back to life this old piece of hardware. Especially, the SSD fixed the ~2 minute-long boot time. This is a good reminder for anyone reading: before buying new crap, spend some time fixing the current crap (some helpful link below). Do your part to save this small stupid planet!!!  

---

References:

- [BIOS Password Backdoors in Laptops](http://dogber1.blogspot.com/2009/05/table-of-reverse-engineered-bios.html) (dogbert)
- [Improving performance](https://wiki.archlinux.org/title/Improving_performance) (archlinux wiki)
- [Optimizing Linux for Slow Computers](https://www.akitaonrails.com/2017/01/17/optimizing-linux-for-slow-computers) (AkitaOnRails)
