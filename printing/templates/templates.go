package templates

// CSStempl is our css template sheet
var CSStempl = []byte(`p {
  margin-bottom: 1.625em;
  font-family: "Arial", Helvetica, sans-serif;
}

h1 {
  font-family: "Arial", Helvetica, sans-serif;
}

ul {
  list-style-type: none;
  margin: 0;
  padding: 0;
  overflow: hidden;
  background-color: #000;
  font-family: "Arial", Helvetica, sans-serif;
}

li {
  float: left;
  border-right: 1px solid #bbb;
}

li:last-child {
  border-right: none;
}

li a {
  display: block;
  color: white;
  text-align: center;
  padding: 14px 16px;
  text-decoration: none;
}

li a:hover {
  background-color: #34C6CD;
}

.active {
  background-color: #A66F00;
  color: white;
}
`)

// ScansTempl is our scans html template
var ScansTempl = `<head>
<link rel="stylesheet" type="text/css" href="../style.css">
</head>

<ul>
  <a href="http://offensivecountermeasures.com/" target="_blank">
    <img src="http://45.33.27.128/wp-content/uploads/2016/02/OCM-logo-022416.png" alt="Offensive Countermeasures"
    style="width:90px; float:left" />
  </a>
  <li><a href="index.html">Home</a></li>
  <li><a href="beacons.html">Beacons</a></li>
  <li><a href="blacklisted.html">Blacklisted</a></li>
  <li><a href="dns.html">DNS</a></li>
  <li><a class="active" href="scans.html">Scans</a></li>
  <li style="float:right">
    <a href="https://github.com/bglebrun/rita" target="_blank">RITA on Github</a>
    </li>
</ul>

<p>
    <h1>RITA: Plaintext output</h1>
  Viewing database: {{.Dbs}}
  <br />

  {{.Writer}}

</p>
`

// Hometempl is our home template html
var Hometempl = `<head>
<link rel="stylesheet" type="text/css" href="./style.css">
</head>

<ul>
    <a href="http://offensivecountermeasures.com/" target="_blank">
      <img src="http://45.33.27.128/wp-content/uploads/2016/02/OCM-logo-022416.png" alt="Offensive Countermeasures"
      style="width:90px; float:left" />
    </a>
  <li style="float:right">
    <a href="https://github.com/bglebrun/rita" target="_blank">RITA on GitHub </a>
    </li>
</ul>

<p>
  <h1>RITA: Plaintext output</h1>

  To view induvidual databases, click on any of the links below.
  <div class="vertical-menu">
    {{range .}}
      <a href="{{.}}/index.html">{{.}}</a>
    {{end}}
  </div>

</p>
`

// DNStempl is our dns page template
var DNStempl = `<head>
<link rel="stylesheet" type="text/css" href="../style.css">
</head>

<ul>
  <a href="http://offensivecountermeasures.com/" target="_blank">
    <img src="http://45.33.27.128/wp-content/uploads/2016/02/OCM-logo-022416.png" alt="Offensive Countermeasures"
    style="width:90px; float:left" />
  </a>
  <li><a href="index.html">Home</a></li>
  <li><a href="beacons.html">Beacons</a></li>
  <li><a href="blacklisted.html">Blacklisted</a></li>
  <li><a class="active" href="dns.html">DNS</a></li>
  <li><a href="scans.html">Scans</a></li>
  <li style="float:right"><a href="https://github.com/bglebrun/rita">RITA on Github</a></li>
</ul>

<p>
    <h1>RITA: Plaintext output</h1>
  Viewing database: {{.Dbs}}
  <br />

  {{.Writer}}

</p>
`

// DBhometempl is our database home template for each directory
var DBhometempl = `<head>
<link rel="stylesheet" type="text/css" href="../style.css">
</head>

<ul>
    <a href="http://offensivecountermeasures.com/" target="_blank">
      <img src="./img/OCM-logo.jpg" alt="Offensive Countermeasures"
      style="width:90px; float:left" />
    </a>
    <li><a class="active" href="index.html">Home</a></li>
    <li><a href="beacons.html">Beacons</a></li>
    <li><a href="blacklisted.html">Blacklisted</a></li>
    <li><a href="dns.html">DNS</a></li>
    <li><a href="scans.html">Scans</a></li>
  <li style="float:right">
    <a href="https://github.com/bglebrun/rita" target="_blank">RITA on GitHub </a>
    </li>
</ul>

<p>
  <h1>RITA: Plaintext output</h1>
  Viewing database: {{.db}}
  <br />
  To view results, click on any of the links above.

</p>
`

// BeaconsTempl is our beacons html template
var BeaconsTempl = `<head>
<link rel="stylesheet" type="text/css" href="../style.css">
</head>

<ul>
  <a href="http://offensivecountermeasures.com/" target="_blank">
    <img src="http://45.33.27.128/wp-content/uploads/2016/02/OCM-logo-022416.png" alt="Offensive Countermeasures"
    style="width:90px; float:left" />
  </a>
  <li><a href="index.html">Home</a></li>
  <li><a class="active" href="beacons.html">Beacons</a></li>
  <li><a href="blacklisted.html">Blacklisted</a></li>
  <li><a href="dns.html">DNS</a></li>
  <li><a href="scans.html">Scans</a></li>
  <li style="float:right">
    <a href="https://github.com/bglebrun/rita" target="_blank">RITA on GitHub </a>
  </li>
</ul>
      <img src="./img/GitHub-Mark-32px.png" style="width:14px" alt="GitHub" /> </a>

<p>
    <h1>RITA: Plaintext output</h1>
  Viewing database: {{.Dbs}}
  <br />

  {{.Writer}}

</p>
`

// BlacklistedTempl is our beacons html template
var BlacklistedTempl = `<head>
<link rel="stylesheet" type="text/css" href="../style.css">
</head>

<ul>
  <a href="http://offensivecountermeasures.com/" target="_blank">
    <img src="http://45.33.27.128/wp-content/uploads/2016/02/OCM-logo-022416.png" alt="Offensive Countermeasures"
    style="width:90px; float:left" />
  </a>
  <li><a href="index.html">Home</a></li>
  <li><a href="beacons.html">Beacons</a></li>
  <li><a class="active" href="blacklisted.html">Blacklisted</a></li>
  <li><a href="dns.html">DNS</a></li>
  <li><a href="scans.html">Scans</a></li>
  <li style="float:right">
    <a href="https://github.com/bglebrun/rita" target="_blank">RITA on GitHub </a>
  </li>
</ul>
      <img src="./img/GitHub-Mark-32px.png" style="width:14px" alt="GitHub" /> </a>

<p>
    <h1>RITA: Plaintext output</h1>
  Viewing database: {{.Dbs}}
  <br />

  {{.Writer}}

</p>
`
