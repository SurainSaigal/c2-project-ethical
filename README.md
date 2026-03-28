## Command and Control Project - Ethical Hacking

# Surain Saigal, Josh Yu

This project is a backdoor which uses this GitHub repo as an intermediary communication point.
This repo is public, but all communication is encrypted using a shared secret key with AES-GCM.
This project is for academic purposes only and is not intended for malicious use.

# Installation

To compile the binary, you must have Go installed.

To compile, run the `build.sh` script in this repo.
Two binaries will be created: `sys_update` and `controller`

Transfer `sys_update` to any directory on the target/victim system, and then run as root.
The binary will auto install itself into `/usr/local/bin/`, create a service to persist across
reboots and failures, and self-delete from its initial location.

Now run `controller` on the command/control system.
You will have a pseudo-shell which acts as a normal root shell on the victim system, granted a bit of time delay.
This shell is not interactive, and won't persist across commands (i.e., you can't run `cd` and still be in the same directory on the next command). You can, however, use this shell to set up an actual interactive reverse shell using netcat
or the like.
