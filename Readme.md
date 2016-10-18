#RITA

Brought to you by Offensive CounterMeasures

###Disclaimer

RITA is not production ready! This software is being released open source as it
is being worked on. The team at OCM (Offensive CounterMeasures) has been
diligently working on the process of seperating the analysis logic from the
front end which is destined to be product for sale by Offensive CounterMeasures.

###Current state

####Terminal output
Right now we're working on getting output that makes getting your analysis data
easy and follows common unix ideas. Ideally this output will eventually be fully
compatible with the formatting of bro's logs which should make working with the
output much easier for those already versed in the useage of tools like bro-cut.
This feature is being worked on.

####Graphical front end
We're also working on getting a minimalist front end to the platform that can
be used to simply avoid the command line. This will probably take longer than
the above and should not be expected to have the data visualization features
present in the AI Hunt project.

###What's here

RITA has all of the logic used to analyze Bro data. With an input of Bro data a
MongoDB database will be created, which can be analyzed for review of that data.
All of the mathematics, lookups, and storage of Offensive CounterMeasures AI
Hunter is available in this package. The only thing not here is the graphical
front end which Offensive CounterMeasures has created to help visualize this
data.

###Installation

1. What you'll need:
 * Bro [https://www.bro.org](https://www.bro.org)
 * MongoDB [https://www.mongodb.com](https://www.mongodb.com)
 * Golang [https://www.golang.org](https://www.golang.org)
 * GNU netcat [http://netcat.sourceforge.net/](http://netcat.sourceforge.net/)

1. Setting up your environment:
  1. Install bro using the directions at [https://www.bro.org/sphinx/install/install.html](https://www.bro.org/sphinx/install/install.html)
  1. Test that bro is working by firing up bro and ensuring that it's spitting out logs. If you're having some trouble
  with bro configuration or use here are some helpful links:
    * Bro quick start [https://www.bro.org/sphinx-git/quickstart/index.html](https://www.bro.org/sphinx-git/quickstart/index.html)
    * broctl [https://www.bro.org/sphinx/components/broctl/README.html](https://www.bro.org/sphinx/components/broctl/README.html)
  1. Install MongoDB (You will need MongoDB 3.2.0 which is not included in the Ubuntu 16.04 package manager. If you use your package manager, make sure it is at least MongoDB version 3.x)
    * Download 3.2.0 at https://www.mongodb.com/download-center?jmp=nav#community
    * Select your version of linux and download the package
  1. Install GNU Netcat, make sure that it is GNU Netcat. NC will not work. [http://netcat.sourceforge.net/](http://netcat.sourceforge.net/)
  1. Install GoLang using the instructions at [https://golang.org/doc/install](https://golang.org/doc/install)
  1. After the install we need to set a local GOPATH for our user. So lets set up a directory in our HomeDir
    * ```mkdir -p $HOME/go/{src,pkg,bin}```
  1. Now we must add the GoPath to our .bashrc file
    * ```echo 'export GOPATH="$HOME/go"' >> $HOME/.bashrc```
  1. We will also want to add our bin folder to the path for this user.
    * ```echo 'export PATH="$PATH:$GOPATH/bin"' >> $HOME/.bashrc```
  1. Load your new configurations with source.
    * ```source $HOME/.bashrc```

1. Getting the sources and building them
  	1. First we want to use the go to grab sources and deps for rita.
    	* ```go get github.com/ocmdev/rita```
  	1. Now lets change to the rita directory.
    	* ```cd $GOPATH/src/github.com/ocmdev/rita```
  	1. Then build rita.
    	* ```go build```
  	1. Now we'll install the rita binary.
  		* ```go install```
  	1. Finally, let's install all of the supporting software.
  		* ```sudo ./install.sh```

1. Configuring MongoDB
  1. If your package manager automatically installs and configures the latest MongoDB 3.x, you can skip this section
  1. Unzip the file you downloaded earlier
    * ```tar -zxvf mongodb-linux-x86_64-[your OS version].tgz```
  1. Copy the directory to it's own folder, this is where the MongoDB process will run
    * ```mkdir -p <path_to_desired_folder>/mongodb && cp -R -n mongodb-linux-x86_64-3.2.10/ <path_to_desired_folder>/mongodb```
  1. Ensure this location is set in your path variable, this can be done quickly with
    * ```echo 'export PATH=<your_mongodb_install_directory>/mongodb-linux-x86_64-3.2.10/bin:$PATH' >> ~/.bashrc```
  1. Load your new bash config
    * ```source $HOME/.bashrc```

1. Launching MongoDB
  1. Again if your package manager automatically installs and configures MongoDB 3.x, you can skip this section
  1. Make your MongoDB directory, usually /data/db
    * ```sudo mkdir -p /data/db```
  1. Then give the user permissions to read/write to our database directory
    * ```sudo chown -R <username> /data```
  1. Now at this point you can watch MongoDB do it's magic before your very eyes with
    * ```mongod```
  1. Otherwise if you're a very busy person like us, you can fork the process as a daemon. Make the log file and grant appropriate permissions
    * ```sudo touch /data/mongod.log && sudo chown <username> mongod.log && sudo chmod u+w```
  1. Then start mongod daemon with
    * ```mongod --fork --logpath /data/mongod.log```
  1. If mongo is still not running, you can check out further documentation at https://docs.mongodb.com/

1. Configuring the system
  1. If you installed as sudo (root) then there will be a default config file at both /usr/local/rita/etc/rita.yaml
  and /etc/rita/config.yaml.
  1. You can also copy the global config from /etc to your homedir and call it .rita. If there's a .rita config that's
  the one that will be used. Here's the order of precendence for configuration.
    	* file given on the command line with the -c flag
    	* $HOME/.rita
    	* /etc/rita/config.yaml
    	* If none of the above files successfully configure the system then the system fails.
  1. You can test a configuration file with ```rita testconfig PATH/TO/FILE``` if the file is syntactically correct rita
  will print the resultant configuration. If it fails an error will be given.
  1. The most important parts of the configuration file are the database path, the path for your netcat binary, a name
  for the database you'd like to create with this dataset, and of course the Bro section of the yaml file which configures
  your parser. There are comments in the yaml file that should help with configuration.

###Getting help
Head over to OFTC and join #ocmdev for any questions you may have. Please
remember that this is an open source project, the developers working in here
have full time jobs and are not your personal tech support. So please be civil
with us.

###License
GNU GPL V3
&copy; Offensive CounterMeasures &trade;

###Contributing

Want to help? We'd love that! Here are some ways to get involved ranging in
difficulty from easiest to hardest.

1. Run the software and tell us when it breaks. We're happy to recieve bug
reports. Just be sure to do the following:
  	* Give very specific descriptions of how to reproduce the bug
  	* Let us know if you're running RITA on wierd hardware
  	* Tell us about the size of the test, the physical resources available, and the

1. Add godoc comments to the code. This software was developed for internal use
mostly on the fly and as needed. This means that the code was not built to the
typical standards of an open source project and we would like to get it there.

1. Fix style compliance issues. Just run golint and start fixing non-compliant
code.

1. Work on bug fixes. Grab from the issues list and submit fixes.

1. Helping add features:
  	* If you'd like to become involved in the development effort please hop on our
OFTC channel at #ocmdev and try and chat with booodead about what's currently
being worked on.
  	* If you have a feature request or idea, also please hop on OFTC #ocmdev and
chat with booodead about your idea. There's a chance we're already working on it and
would be happy to share that work with you.

#####Submitting work:
Please send pull requests and such as small as possible. As this is a product that
we use internally, as well as a backend for a piece of commercially supported
software. Every line of code that goes in must be inspected and approved. So if it
is taking a while to get back to you on your work, or we reject code, don't be
offended, we're just paranoid and desire to get this project to a very stable and
useable place.
