package templates

// CSStempl is our css template sheet
var CSStempl = []byte(`p {
  margin-bottom: 1.625em;
  font-family: 'Lucida Sans', Arial, sans-serif;
}

p {
  font-family: 'Lucida Sans', Arial, sans-serif;
  text-indent: 30px;
}

h1 {
  color: #000;
  font-family: 'Lato', sans-serif;
  font-size: 32px;
  font-weight: 300;
  line-height: 58px;
  margin: 0 0 58px;
  text-indent: 30px;
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

div {
  color: #adb7bd;
  font-family: 'Lucida Sans', Arial, sans-serif;
  font-size: 16px;
  line-height: 26px;
  margin: 0;
}

li a:hover {
  background-color: #34C6CD;
}

.vertical-menu {
  width: 150px;
}

.vertical-menu a {
  background-color: #000;
  color: white;
  display: block;
  padding: 12px;
  text-decoration: none;
  text-align: center;
  vertical-align: middle;
}

.vertical-menu a:hover {
  background-color: #34C6CD;
}

.active {
  background-color: #A66F00;
  color: white;
}

.info {
  padding: 30px 40px;
  font-family: "Arial", Helvetica, sans-serif;
  font-size: 18px;
  border-left: 5px solid #000;
  margin: 20px 40px;
  color: #a9a9a9;
}
`)

// ScansTempl is our scans html template
var ScansTempl = `<head>
<meta content="text/html;charset=utf-8" http-equiv="Content-Type">
<meta content="utf-8" http-equiv="encoding">
<link rel="stylesheet" type="text/css" href="../style.css">
<script src="https://use.fontawesome.com/96648b06fb.js"></script>
</head>

<ul>
  <a href="http://offensivecountermeasures.com/" target="_blank">
    <img src="http://45.33.27.128/wp-content/uploads/2016/02/OCM-logo-022416.png" alt="Offensive Countermeasures"
    style="width:90px; float:left" />
  </a>
  <li><a href="../index.html">RITA</a></li>
  <li><a href="index.html">Viewing: {{.Dbs}}</a></li>
  <li><a href="beacons.html">Beacons</a></li>
  <li><a href="blacklisted.html">Blacklisted</a></li>
  <li><a href="dns.html">DNS</a></li>
  <li><a class="active" href="scans.html">Scans</a></li>
  <li style="float:right">
    <a href="https://github.com/bglebrun/rita" target="_blank">RITA on
      <i class="fa fa-github fa-lg" aria-hidden="true" alt="GitHub"></i>
    </a>
  </li>
</ul>

<p>

  {{.Writer}}

</p>
`

// Hometempl is our home template html
var Hometempl = `<head>
<meta content="text/html;charset=utf-8" http-equiv="Content-Type">
<meta content="utf-8" http-equiv="encoding">
<script src="https://use.fontawesome.com/96648b06fb.js"></script>
<link rel="stylesheet" type="text/css" href="./style.css">
</head>

<ul>
    <a href="http://offensivecountermeasures.com/" target="_blank">
      <img src="http://45.33.27.128/wp-content/uploads/2016/02/OCM-logo-022416.png" alt="Offensive Countermeasures"
      style="width:90px; float:left" />
    </a>
  <li><a class="active" href="../index.html">RITA</a></li>
  <li style="float:right">
    <a href="https://github.com/bglebrun/rita" target="_blank">RITA on
      <i class="fa fa-github fa-lg" aria-hidden="true" alt="GitHub"></i>
    </a>
  </li>
</ul>

<p>
  <div class="info">To view induvidual databases, click on any of the links below.</div>
  <div class="vertical-menu">
    {{range .}}
      <a href="{{.}}/index.html">{{.}}</a>
    {{end}}
  </div>

</p>
`

// DNStempl is our dns page template
var DNStempl = `<head>
<meta content="text/html;charset=utf-8" http-equiv="Content-Type">
<meta content="utf-8" http-equiv="encoding">
<script src="https://use.fontawesome.com/96648b06fb.js"></script>
<link rel="stylesheet" type="text/css" href="../style.css">
</head>

<ul>
  <a href="http://offensivecountermeasures.com/" target="_blank">
    <img src="http://45.33.27.128/wp-content/uploads/2016/02/OCM-logo-022416.png" alt="Offensive Countermeasures"
    style="width:90px; float:left" />
  </a>
  <li><a href="../index.html">RITA</a></li>
  <li><a href="index.html">Viewing: {{.Dbs}}</a></li>
  <li><a href="beacons.html">Beacons</a></li>
  <li><a href="blacklisted.html">Blacklisted</a></li>
  <li><a class="active" href="dns.html">DNS</a></li>
  <li><a href="scans.html">Scans</a></li>
  <li style="float:right">
    <a href="https://github.com/bglebrun/rita" target="_blank">RITA on
      <i class="fa fa-github fa-lg" aria-hidden="true" alt="GitHub"></i>
    </a>
  </li>
</ul>

<p>

  {{.Writer}}

</p>
`

// DBhometempl is our database home template for each directory
var DBhometempl = `<head>
<meta content="text/html;charset=utf-8" http-equiv="Content-Type">
<meta content="utf-8" http-equiv="encoding">
<script src="https://use.fontawesome.com/96648b06fb.js"></script>
<link rel="stylesheet" type="text/css" href="../style.css">
</head>

<ul>
  <a href="http://offensivecountermeasures.com/" target="_blank">
    <img src="http://45.33.27.128/wp-content/uploads/2016/02/OCM-logo-022416.png" alt="Offensive Countermeasures"
    style="width:90px; float:left" />
  </a>
  <li><a href="../index.html">RITA</a></li>
  <li><a class="active" href="index.html">Viewing: {{.Dbs}}</a></li>
  <li><a href="beacons.html">Beacons</a></li>
  <li><a href="blacklisted.html">Blacklisted</a></li>
  <li><a href="dns.html">DNS</a></li>
  <li><a href="scans.html">Scans</a></li>
  <li style="float:right">
    <a href="https://github.com/bglebrun/rita" target="_blank">RITA on
      <i class="fa fa-github fa-lg" aria-hidden="true" alt="GitHub"></i>
    </a>
  </li>
</ul>

<p>
  <div>To view results, click on any of the links above.</div>

</p>
`

// BeaconsTempl is our beacons html template
var BeaconsTempl = `<head>
<meta content="text/html;charset=utf-8" http-equiv="Content-Type">
<meta content="utf-8" http-equiv="encoding">
<link rel="stylesheet" type="text/css" href="../style.css">
<script src="https://use.fontawesome.com/96648b06fb.js"></script>
</head>

<ul>
  <a href="http://offensivecountermeasures.com/" target="_blank">
    <img src="http://45.33.27.128/wp-content/uploads/2016/02/OCM-logo-022416.png" alt="Offensive Countermeasures"
    style="width:90px; float:left" />
  </a>
  <li><a href="../index.html">RITA</a></li>
  <li><a href="index.html">Viewing: {{.Dbs}}</a></li>
  <li><a class="active" href="beacons.html">Beacons</a></li>
  <li><a href="blacklisted.html">Blacklisted</a></li>
  <li><a href="dns.html">DNS</a></li>
  <li><a href="scans.html">Scans</a></li>
  <li style="float:right">
    <a href="https://github.com/bglebrun/rita" target="_blank">RITA on
      <i class="fa fa-github fa-lg" aria-hidden="true" alt="GitHub"></i>
    </a>
  </li>
</ul>

<p>

  {{.Writer}}

</p>
`

// BlacklistedTempl is our beacons html template
var BlacklistedTempl = `<head>
<meta content="text/html;charset=utf-8" http-equiv="Content-Type">
<meta content="utf-8" http-equiv="encoding">
<link rel="stylesheet" type="text/css" href="../style.css">
<script src="https://use.fontawesome.com/96648b06fb.js"></script>
</head>

<ul>
  <a href="http://offensivecountermeasures.com/" target="_blank">
    <img src="http://45.33.27.128/wp-content/uploads/2016/02/OCM-logo-022416.png" alt="Offensive Countermeasures"
    style="width:90px; float:left" />
  </a>
  <li><a href="../index.html">RITA</a></li>
  <li><a href="index.html">Viewing: {{.Dbs}}</a></li>
  <li><a href="beacons.html">Beacons</a></li>
  <li><a class="active" href="blacklisted.html">Blacklisted</a></li>
  <li><a href="dns.html">DNS</a></li>
  <li><a href="scans.html">Scans</a></li>
  <li style="float:right">
    <a href="https://github.com/bglebrun/rita" target="_blank">RITA on
      <i class="fa fa-github fa-lg" aria-hidden="true" alt="GitHub"></i>
    </a>
  </li>
</ul>

<p>

  {{.Writer}}

</p>
`
