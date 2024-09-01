# Tailnode

BubbleTea TUI to manage Tailscale/Mullvad VPN Exit Nodes

## Why?

Currently, the Tailscale Windows GUI forces you to scroll through a giant list of Mullvad VPN hostnames in order to choose an exit node. This user experience is subpar compared to other OS's and is being actively worked on by the Tailscale team.

In the meantime, I wanted a better experience for myself. I also wanted to explore making TUIs using [Bubbletea](https://github.com/charmbracelet/bubbletea).

Tailnode is packaged as an exe but using the magic of `CommandContext`, invokes a terminal window and loads the TUI within this window. This simplifies the experience by letting users run the program without having to know what a terminal is or how to run a `go` program.

## Features

Tailnode aims to be fast, simple and ✨*cute*✨

![image](https://github.com/user-attachments/assets/bdb6fcb3-825d-44b2-bcd1-27c57c88575e)

Tailnode allows you to
- View all exit nodes (IPs, Hostnames, Country & City) as a well-formatted table
- Set/Unset exit nodes via table
- Filter exit nodes by Country or City

More features may come in the future but I think Tailnode in its current iteration solves my pressing problem :)

## Links

- [go-winres](https://github.com/tc-hib/go-winres)
- [FR: Improve Mullvad exit-node selection lists on Windows and Android](https://github.com/tailscale/tailscale/issues/9421)
