# System Requirements

* Operating System - The preferred platform is 64-bit Ubuntu 18.04 LTS. The system should be patched and up to date using apt-get.
  * The automated installer will also support Security Onion and CentOS 7. You can install on other operating systems using [docker](Docker%20Usage.md) or our [manual installation](Manual%20Installation.md).

If RITA is used on a separate system from Zeek our recommended specs are:
* Processor - Two or more cores. RITA uses parallel processing and benefits from more CPU cores.
* Memory - 16GB. Larger datasets may require more memory.
* Storage - RITA's datasets are significantly smaller than the Zeek logs so storage requirements are minimal compared to retaining the Zeek log files.


## Zeek
The following requirements apply to the Zeek system.

* Processor - Two cores plus an additional core for every 100 Mb of traffic being captured. (three cores minimum). This should be dedicated hardware, as resource congestion with other VMs can cause packets to be dropped or missed.
* Memory - 16GB minimum. 64GB if monitoring 100Mb or more of network traffic. 128GB if monitoring 1Gb or more of network traffic.
* Storage - 300GB minimum. 1TB or more is recommended to reduce log maintenance.
* Network - In order to capture traffic with Zeek, you will need at least 2 network interface cards (NICs). One will be for management of the system and the other will be the dedicated capture port. Intel NICs perform well and are recommended.
