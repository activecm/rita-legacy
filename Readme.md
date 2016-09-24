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

1. Clone the repo.
1. run '''go get'''
1. run '''go install'''
1. run sudo ./install.sh
1. edit /etc/rita/config.yaml to contain the address and port of your mongodb
server, a name for a database you'd like to build, where to find the bro logs etc.
1. run rita --help to view available commands

###Getting help
Head over to OFTC and join #ocmdev for any questions you may have. Please 
remember that this is an open source project, the developers working in here
have full time jobs and are not your personal tech support. So please be civil
with us.

###License 
GNU GPL V3
&copy; Offensive CounterMeasures &trade;

###Contributing

Want to get help? We'd love that! Here are some ways to get involved ranging in
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

