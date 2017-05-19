package templates

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
  <li><a class="active" href="index.html">Viewing: {{.}}</a></li>
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
