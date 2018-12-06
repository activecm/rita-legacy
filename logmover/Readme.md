# Move Logs From Separate Bro IDS/ Zeek Systems To RITA

If you would like to run Bro IDS/ Zeek on a separate system from RITA,
the scripts in this directory will help you automatically transfer the logs
collected by Bro IDS/ Zeek to your RITA installation and kick off the
RITA import and analysis process.

The two scripts work with each other. [Watcher.sh](./watcher.sh) runs on the
RITA system, waits for data to finish transferring in, and kicks off the
import and analyze process. On the other hand, [logpush.bash](./logpush.bash)
runs on a Bro IDS/ Zeek system, transfers the previous day's logs to the
RITA system, and lets the watcher script know when it finishes.

Multiple instances of Bro IDS/ Zeek instances may be used with a single instance
of RITA by running an instance of [logpush.bash](./logpush.bash) on each
Bro IDS/ Zeek system.

## Installation

1. Ensure that each system running Bro IDS/ Zeek is able to access the system running RITA over SSH using an SSH key that is not password protected.
    - Remember the SSH connection details, we will need them later.
        - RITA system hostname/ IP address
        - The name of the user account used to access the RITA system
        - The path to the SSH key used to access the RITA system
1. Install [watcher.sh](./watcher.sh) on the RITA system
    - Edit [watcher.sh](./watcher.sh)
        - Set `LOG_DIR` to the directory you intend to keep your Bro logs in on the RITA system
    - Copy the edited script to the RITA system
        - This guide assumes the watcher script is placed at `/usr/local/bin/watcher.sh`
    - Ensure the script is executable
        - `sudo chmod 755 /usr/local/bin/watcher.sh`
    - If the script is placed in `/usr/local/bin`, ensure `root` owns the script
        - `sudo chown root:root /usr/local/bin/watcher.sh`
    - Ensure the directory referenced by `LOG_DIR` exists on the RITA system
    - As the user noted above, run `crontab -e`
        - This guide sets [watcher.sh](./watcher.sh) to run at 12:10 a.m.
        - Add `10 0 * * * /usr/local/bin/watcher.sh` to the end of the user's crontab
1. Install [logpush.bash](./logpush.bash) on each Bro IDS/ Zeek System
    - Edit [logpush.bash](./logpush.bash)
        - Set `USER` to the name of the user account determined in the first step
        - Set `REMOTE` to the hostname/ IP address of the RITA system
        - Set `REMOTE_LOG_DIR` to the same value as `LOG_DIR` in the second step
        - Set `LOCAL_LOG_DIR` to the directory containing your Bro IDS/ Zeek logs
        - Set `COLLECTOR` the name of this Bro IDS/ Zeek System. This will be used to name the RITA datasets which originate from this system.
        - Set `KEYFILE` to the path of the SSH key that will be used to connect to the RITA system
    - Copy the edited script to the Bro IDS/ Zeek system
        - This guide assumes the watcher script is placed at `/usr/local/bin/logpush.bash`
    - Ensure the script is executable
        - `sudo chmod 755 /usr/local/bin/logpush.bash`
    - If the script is placed in `/usr/local/bin`, ensure `root` owns the script
        - `sudo chown root:root /usr/local/bin/logpush.bash`
    - As a user with access to the Bro logs and SSH key, run `crontab -e`
        - This guide sets [logpush.bash](./logpush.bash) to run at 12:05 a.m.
        - It is important that [logpush.bash](./logpush.bash) runs before [watcher.sh](./watcher.sh). However, [logpush.bash](./logpush.bash) does *not* have to finish executing before [watcher.sh](./watcher.sh) runs.
        - Add `5 0 * * * /usr/local/bin/logpush.bash` to the end of the user's crontab

If all goes well, logs will be transferred from the Bro IDS/ Zeek box at 12:05 a.m. The watcher script will kick off at 12:10 a.m., wait for the transfers to finish, and begin analyzing the data.

NOTE: There is a bug at the moment which requires [logpush.bash](./logpush.bash) to be able to create a file in the working directory. If [logpush.bash](./logpush.bash) is placed in `/usr/local/bin` the script must be ran as root. A patch will soon be available which will create the needed file in a `/tmp` directory.

NOTE: [logpush.bash](./logpush.bash) and [watcher.sh](./watcher.sh) will not work if the directory referenced by
`LOG_DIR`/ `REMOTE_LOG_DIR` is stored within a NFS filesystem. NFS does not properly support `flock`.
