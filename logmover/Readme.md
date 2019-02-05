# Move Logs From Separate Bro IDS/ Zeek Systems To RITA

If you would like to run Bro IDS/ Zeek on a separate system from RITA,
the scripts in this directory will help you automatically transfer the logs
collected by Bro IDS/ Zeek to your RITA installation and kick off the
RITA import and analysis process.

The two scripts work with each other. [log_watcher.bash](./log_watcher.bash) runs on the
RITA system, waits for data to finish transferring in, and kicks off the
import and analyze process. On the other hand, [log_pusher.bash](./log_pusher.bash)
runs on a Bro IDS/ Zeek system, transfers the previous day's logs to the
RITA system, and lets the watcher script know when it finishes.

Multiple instances of Bro IDS/ Zeek instances may be used with a single instance
of RITA by running an instance of [log_pusher.bash](./log_pusher.bash) on each
Bro IDS/ Zeek system.

NOTE: [log_watcher.bash](./log_watcher.bash) deletes the received logs after they are
imported. This prevents duplicating data archived on the Bro IDS/ Zeek collector.

## Installation

1. Ensure that each system running Bro IDS/ Zeek is able to access the system running RITA over SSH using an SSH key that is not password protected.
    - Remember the SSH connection details, you will need them later.
        - RITA system hostname/ IP address
        - The name of the user account used to access the RITA system
        - The path to the SSH key used to access the RITA system
1. Install [log_watcher.bash](./log_watcher.bash) on the RITA system
    - Edit [log_watcher.bash](./log_watcher.bash)
        - Set `LOG_DIR` to the directory RITA is set to read Bro logs from
    - Copy the edited script to the RITA system
        - This guide assumes the watcher script is placed at `/usr/local/bin/log_watcher.bash`
    - Ensure the script is executable
        - `sudo chmod 755 /usr/local/bin/log_watcher.bash`
    - If the script is placed in `/usr/local/bin`, ensure `root` owns the script
        - `sudo chown root:root /usr/local/bin/log_watcher.bash`
    - Ensure the directory referenced by `LOG_DIR` exists on the RITA system
    - As the user noted above, run `crontab -e`
        - This guide sets [log_watcher.bash](./log_watcher.bash) to run at 12:10 a.m.
        - Add `10 0 * * * /usr/local/bin/log_watcher.bash` to the end of the user's crontab
1. Install [log_pusher.bash](./log_pusher.bash) on each Bro IDS/ Zeek System
    - Edit [log_pusher.bash](./log_pusher.bash)
        - Set `USER` to the name of the user account determined in the first step
        - Set `REMOTE` to the hostname/ IP address of the RITA system
        - Set `REMOTE_LOG_DIR` to the same value as `LOG_DIR` in the second step
        - Set `LOCAL_LOG_DIR` to the directory containing your Bro IDS/ Zeek logs
        - Set `COLLECTOR` to the name of this Bro IDS/ Zeek System. This will be used to name the RITA datasets which originate from this system.
        - Set `KEYFILE` to the path of the SSH key that will be used to connect to the RITA system
    - Copy the edited script to the Bro IDS/ Zeek system
        - This guide assumes the watcher script is placed at `/usr/local/bin/log_pusher.bash`
    - Ensure the script is executable
        - `sudo chmod 755 /usr/local/bin/log_pusher.bash`
    - If the script is placed in `/usr/local/bin`, ensure `root` owns the script
        - `sudo chown root:root /usr/local/bin/log_pusher.bash`
    - As a user with access to the Bro logs and SSH key, run `crontab -e`
        - This guide sets [log_pusher.bash](./log_pusher.bash) to run at 12:05 a.m.
        - It is important that [log_pusher.bash](./log_pusher.bash) runs before [log_watcher.bash](./log_watcher.bash). However, [log_pusher.bash](./log_pusher.bash) does *not* have to finish executing before [log_watcher.bash](./log_watcher.bash) runs.
        - Add `5 0 * * * /usr/local/bin/log_pusher.bash` to the end of the user's crontab

If all goes well, logs will be transferred from the Bro IDS/ Zeek box at 12:05 a.m. The watcher script will kick off at 12:10 a.m., wait for the transfers to finish, and begin analyzing the data.


NOTE: [log_pusher.bash](./log_pusher.bash) and [log_watcher.bash](./log_watcher.bash) will not work if the directory referenced by
`LOG_DIR`/ `REMOTE_LOG_DIR` is stored within a NFS filesystem. NFS does not properly support `flock`.
