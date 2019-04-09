# Mongo Configuration for RITA

This is the default MongoDB configuration section in RITA's configuration file `/etc/rita/config.yaml`.

```yaml
MongoDB:
    # See https://docs.mongodb.com/manual/reference/connection-string/
    ConnectionString: mongodb://localhost:27017
    # Example with authentication. Be sure to change the AuthenticationMechanism as well. 
    # ConnectionString: mongodb://username:password@localhost:27017

    # Accepted Values: null, "SCRAM-SHA-1", "MONGODB-CR", "PLAIN"
    # Since Mongo version 3.0 the default authentication mechanism is SCRAM-SHA-1
    AuthenticationMechanism: null

    # The time in hours before RITA's connection to MongoDB times out. 0 waits indefinitely.
    SocketTimeout: 2

    # For encrypting data on the wire between RITA and MongoDB
    TLS:
        Enable: false
        #If set, RITA will verify the MongoDB certificate's hostname and validity
        VerifyCertificate: false
        #If set, RITA will use the provided CA file instead of the system's CA's
        CAFile: null
```


## Connection String

The basic connection string format for RITA is:

```
mongodb://[username:password@]host1[:port1]
```

References:
- https://docs.mongodb.com/manual/reference/connection-string/


## Authentication

Possible Values:
* null (default)
* SCRAM-SHA-1 (preferred)
* MONGODB-CR (not tested) 
* PLAIN (not tested)
* x509 (not supported)

To configure MongoDB with authentication, you need to create a user and configure Mongod to require authentication for connections. Then configure RITA to authenticate with the user you created.


### Create a Mongo User

Mongo allows you to enable authentication before or after creating a user. But for simplicity it is easier to create the user first.

Connect to your mongo instance by running the `mongo` client. By default it will connect to `127.0.0.1` on port `27017` so if you haven't change the defaults you can just run:

```
mongo
```

And you should be greeted with:

```
MongoDB shell version v3.4.10
connecting to: mongodb://127.0.0.1:27017
MongoDB server version: 3.4.10
> 
```

Next, enter the following command to create a user, replacing `user` and `pwd` values with your desired values. The following example creates a user with the username "rita" and the password "assumebreach". The "userAdminAnyDatabase" role used here is a built-in Superuser level role. To read more about the different built-in roles available visit the link in the references below.

```
db.getSiblingDB('admin').createUser(
  {
    user: "rita",
    pwd: "assumebreach",
    roles: [ { role: "userAdminAnyDatabase", db: "admin" } ]
  }
)
```

And finally, exit the mongo shell.

```
exit
```

References
- https://docs.mongodb.com/manual/tutorial/enable-authentication/
- https://docs.mongodb.com/manual/reference/built-in-roles/


### Mongo Config

In a default Mongo installation, authentication is disabled. In older versions of Mongo, this was a serious security vulnerability because Mongo also defaulted to listening on all network interfaces. This meant that remote systems could access the databases without authentication. Since version 3.6, Mongo will only listen on localhost by default. This means that only clients connecting from the same system will be able to connect.

The version of Mongo that RITA installs has authentication disabled and listens on the localhost interface only. If you wish to use RITA and Mongo on separate systems we recommend you enable both authentication and encryption.

To enable authentication, edit the Mongo config file. Ubuntu's default location for this file is `/etc/mongod.conf`. Since version 3.0 of Mongo, the default authentication mechanism (when enabled) is "SCRAM-SHA-1". 

Add or modify your security section of the config file to include authorization.

```yaml
security:
  authorization: enabled
```

Restart your Mongod service for the changes to take effect.

```
service mongod restart
```


References:
- https://docs.mongodb.com/manual/core/authentication/
- https://docs.mongodb.com/manual/reference/configuration-options/#security-options


### RITA Config

To enable authentication and provide a username and password in RITA, modify the connection string in RITA's config file (`/etc/rita/config.yaml`). 

Of the possible values for `AuthenticationMechanism`, the only officially supported values are `null` or `SCRAM-SHA-1`.

This example configures RITA to authenticate with a username of "rita" and a password of "assumebreach" and to use "SCRAM-SHA-1" for the authentication protocol.

```yaml
MongoDB:
  ConnectionString: mongodb://rita:assumebreach@localhost:27017
  AuthenticationMechanism: SCRAM-SHA-1
```

You can test that RITA is configured correctly by running `rita show-databases`.

If the connection is successful, RITA will show the list of databases (or no output if you do not have any databases imported yet).

If authentication is configured incorrectly, RITA will give the following output:

```
rita show-databases
Failed to connect to database: server returned error on SASL authentication step: Authentication failed.
```


## Encryption

Possible Values:
* None (default)
* TLS
  * Self-Signed Certificate
  * Trusted Certificate
  * Certificate Verification

Mongo's method of encrypting connnections is to use TLS/SSL. But by default Mongo does not have encryption enabled on client connections. 

To quote the Mongo documentation:

> Before you can use SSL, you must have a .pem file containing a public key certificate and its associated private key.

> MongoDB can use any valid SSL certificate issued by a certificate authority or a self-signed certificate. If you use a self-signed certificate, although the communications channel will be encrypted, there will be no validation of server identity. Although such a situation will prevent eavesdropping on the connection, it leaves you vulnerable to a man-in-the-middle attack. Using a certificate signed by a trusted certificate authority will permit MongoDB drivers to verify the server’s identity.

Source: https://docs.mongodb.com/manual/core/security-transport-encryption/


### Mongo Config

The basic Mongo config file to enable encryption is shown below (`/etc/mongod.conf` on Ubuntu). This `net` configuration will listen on port `27017` (default) and only listen on the local interface `127.0.0.1` (default). If you want to allow remote connections you will need to change the `bindIp` to either `0.0.0.0` for all interfaces or the IP of the specific interface you want to listen on.

The `ssl` portion of the configuration tells Mongo to _only_ accept encrypted connections. The `requireSSL` setting will refuse any unencrypted connections. The `PEMKeyFile` is the path to the file mentioned above in the quote from Mongo docs. Generating or obtaining this file will be covered below.

```yaml
net:
  port: 27017
  bindIp: 127.0.0.1
  ssl:
    mode: requireSSL
    PEMKeyFile: /etc/ssl/mongodb-cert.pem
```

Restart your Mongod service for the changes to take effect.

```
service mongod restart
```

References:
- https://docs.mongodb.com/manual/reference/configuration-options/#net-options


### RITA Config

The following RITA configuration (`/etc/rita/config.yaml`) snippet is sufficient to enable encrypted communication. Please note that while encryption and authentication are often used together, they are independent settings. The authentication settings aren't shown here but can be added.

Note: RITA does not support the common `?ssl=true` option on Mongo's connection string to enable encryption. You must use the `TLS` section of RITA's config file.

```yaml
MongoDB:
  TLS:
    Enable: true
```

Please make sure you understand the different options for certificates and validation detailed below, as well as the potential for man-in-the-middle attacks if configured incorrectly, before exposing Mongo to an untrusted network.


### Certificates

As stated by the Mongo documentation, you can either obtain a certificate signed by a trusted authority or generate your own self-signed certificate.


#### Self-Signed

There are a great many options when generating a self-signed certificate. The following command will generate a private key (mongodb-cert.key) and a public key (mongodb-cert.crt) in x509 format using the RSA algorithm with a 2048-bit key. This certificate will expire 5 years (1825 days) from the time it is generated.

```
openssl req -x509 -newkey rsa:2048 -days 1825 -nodes -out mongodb-cert.crt -keyout mongodb-cert.key
```

Openssl will then prompt you for several pieces of information. You may fill most of this in with artibrary values that are appropriate to you. But the `Common Name` value is important as it will be used in certificate verification. Set this value to the remote hostname of your Mongo server (i.e. The hostname you will put in your RITA `ConnectionString` config). RITA must be able to reach your Mongo server using this hostname. 

```
Generating a 2048 bit RSA private key
............................................+++
............+++
writing new private key to 'mongodb-cert.key'
-----
You are about to be asked to enter information that will be incorporated
into your certificate request.
What you are about to enter is what is called a Distinguished Name or a DN.
There are quite a few fields but you can leave some blank
For some fields there will be a default value,
If you enter '.', the field will be left blank.
-----
Country Name (2 letter code) [AU]:US
State or Province Name (full name) [Some-State]:
Locality Name (eg, city) []:
Organization Name (eg, company) [Internet Widgits Pty Ltd]:
Organizational Unit Name (eg, section) []:
Common Name (e.g. server FQDN or YOUR name) []:localhost
Email Address []:
```

This will leave you with two files: `mongodb-cert.key` and `mongodb-cert.crt`. In order to put them in the .pem file format that Mongo expects, simply concatenate the two files together like so:

```
cat mongodb-cert.key mongodb-cert.crt > mongodb-cert.pem
```

References:
- https://docs.mongodb.com/manual/tutorial/configure-ssl/


#### Trusted

Obtaining a trusted certificate is beyond the scope of this document. See the Verification secion below for details on configuring RITA to use a trusted certificate.

References:
- https://letsencrypt.org/


#### Verification

By default, RITA will not validate a certificate's authenticity. This is not ideal as it leaves connections open to man-in-the-middle attacks on untrusted networks. 

RITA's `VerifyCertificate` option will validate two things:
1) The certificate is correctly signed by a trusted authority. The trusted authorities are determined by the system's CA store or by specifying a path to a `CAFile` in RITA's config.
2) The `CN` (aka Common Name) field in the certificate must match the hostname where RITA is connecting. That is, the value must be the same as the hostname used in the `ConnectionString`. **This cannot be an IP address.**

Once you have RITA configured, you can test your configuration by running `rita show-databases`.

If the connection is successful, RITA will show the list of databases (or no output if you do not have any databases imported yet).

If encryption or certificate verification is configured incorrectly, RITA will give the following output:

```
rita show-databases
Failed to connect to database: no reachable servers
```

##### Trusted Certificate Verification Example

The following example configuration assumes your Mongo server is located at `activecountermeasures.com` and that you have obtained and configured Mongo with a certificate signed with a valid certificate authority. In this case, you do not need to specify a `CAFile` path.

```yaml
MongoDB:
  ConnectionString: mongodb://activecountermeasures.com:27017
  TLS:
    Enable: true
    VerifyCertificate: true
```

##### Self-Signed Certificate Verification Example

In order to validate a self-signed certificate, you must specify the path to the CA file (commonly with a .crt extension). If you followed this document to generate one then it will be named `mongodb-cert.crt`. RITA does not need the private key (.key), or the combined file (.pem) which also contains the private key. You should protect the private key and not copy it anywhere unnecessarily.

We recommend putting your certificate file at `~/.rita/mongodb-cert.crt`, which is used in the example below. The hostname used for the connection in this case is `localhost` and thus when you generated your certificate you must match this hostname.

```yaml
MongoDB:
  ConnectionString: mongodb://localhost:27017
  TLS:
    Enable: true
    VerifyCertificate: true
    CAFile: $HOME/.rita/mongodb-cert.crt
```

Note: `~` does not expand in RITA's config file and will cause an error. Use `$HOME` instead.


##### Self-Signed Certificate Verification Invalid Example

The following shows one example of a configuration that will not work. This is because an IP address (`127.0.0.1`) is used in the `ConnectionString`. Even if you set `CN` to `127.0.0.1` or add `IP:127.0.0.1` as a `SAN` when generating your certificate, this will still fail to validate.

```yaml
MongoDB:
  ConnectionString: mongodb://127.0.0.1:27017
  TLS:
    Enable: true
    VerifyCertificate: true
    CAFile: $HOME/.rita/mongodb-cert.crt
```


## Complete Example with Authentication and Encryption

For completeness, here is an example of RITA's `MongoDB` config section configured for authentication (username "rita" and password "assumebreach") and encryption (self-signed certificate with validation located at "localhost").

```yaml
MongoDB:
  # See https://docs.mongodb.com/manual/reference/connection-string/
  ConnectionString: mongodb://rita:assumebreach@localhost:27017
  # How to authenticate to MongoDB
  # Accepted Values: null, "SCRAM-SHA-1", "MONGODB-CR", "PLAIN"
  AuthenticationMechanism: SCRAM-SHA-1
  # The time in hours before RITA's connection to MongoDB times out. 0 waits indefinitely.
  SocketTimeout: 2
  # For encrypting data on the wire between RITA and MongoDB
  TLS:
    Enable: true
    #If set, RITA will verify the MongoDB certificate's hostname and validity
    VerifyCertificate: true
    #If set, RITA will use the provided CA file instead of the system's CA's
    CAFile: $HOME/.rita/mongodb-cert.crt
```
