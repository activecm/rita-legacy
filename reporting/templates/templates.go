package templates

import "html/template"

//ReportingInfo fills the templates listed in html/template
type ReportingInfo struct {
	DB     string
	Writer template.HTML
}

var activecmImg = "<img src=\" data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAKcAAABwCAYAAAB7LWB7AAAAAXNSR0IArs4c6QAAAAlwSFlzAAAYmwAAGJsBSXWDlAAAFFVJREFUeAHtXQl0HMWZruqekWTJFzaSE4MBw0KS5+c4ib1OwHmLjYO9vkYjlsvk7WLyEgIaWWAnYXF0pKMjDss+TKzjBXKwy5GwqyU6bGxwWKwNOA6JIYHEOdgQAjg2PrDJymB51N21X41mpO7RHD3d0z1tufpJr+v866+v/vnr+quaEPEIBAQCAgGBgEBAICAQEAgIBAQCAgGBgEBAIOAvBKi/2Dnzuals7fgniZArRmrC2M7u+preEb9wWEYgYDmlSGgJAcrIEkLputHE9B24hXCOAmLZhR+5eAQC/kRAaE5/tktarubfemtw5sw5H6QkOFUKqCVUl+S0idNEqJLOKKGqpuqDQyR6ZJfylaNIytIkL1iwEM6CQW+t4GVfvrdswuSSSipLKwijf0sJuxTDhniPh+bLWTQJCZDh7HJQJkUkSKpaOgcZYb8hlOwhKtvWw47tJoqiW+PQvVRCON3D1hHlsLJlKgkUbYIg3o5Z66QYsdj01YU5LCUl0KQLUMYCEqB3hFn566S5o7VHO/pQIYXUV8IZbulYi1/vtY5aNU3m0ye123ZuruXdl++f0Nc7lkC9PQqBmVkIZimls9EO362SKm4ZVLbeuFOpPVAIPnwlnAAlAhAWuQFEUWlgO+g+5AbtfNIMt7T9A3D4IWgGU9FljLyP4eEblJIhdPPlGDyWpEqXNYyxkxD+40hXiv8LIYxFKfIsKgnKe8PNWxf3NNS+liLe1SDfCGfVpvbpqOmn3KqtRNka0Pa1cIZa2hYQKj0GPk2CyRhTEfYwZizf6325/wXS1aXlE6fFilIyNTh9CWFyNYR+tZk2PZ9I8o7FSsf8fiVy0hznrs83wqlPpKswTLcxvLcGEDTMshXrtxbvbKs9bS2Ht6kgIAGJSA9DmxWbSmZsP9P1tb2N639tCs+jp19RBkFuJ/8PN3csg+Z+BJq0IlEEeLrsnCD5Jvw1iTAv3r5Z5wQjXLO59gDgsqIKaalrBTgkPCV47joIxUeMZKAx9w0MDCxyUzCN5XF3T0NkV1Qll2P2ftAYB16+uErpvNgY5rbbF8LJ1+4wllrmemUlGnK7DLv0JSaZtBKE451TNBp65p67/2qXpt18TyrVfyJMq4oPJ2Jk8MMJBGRym12advL5QjhnzZq3BJWfbKcCOeVhNGk8lVNu1xKHmu//ELrRecYCmE7qnq7bcMgY5qW7p77252iTB4xlYtx+vdHvttsXwomGcbVLHwGRkvPCTZ3zR/w+cVAS+IyJFUaOR49o/2YKK4AnOkTuwyQMf/GH0gtDSsffJLxuv/0hnISscruiCfpM0n3XtUNDfSLBH39j8vaUHyZuw907do4MjyQxE6+GqLw7Cy6clU1tc9E4s/NeszQEMTHynXBiF+hSE7uMvWzyF9JDiYkXKifx6iJvBRdOIlFvuvQ4iPghfKyypWOWi5jaIM0+aMwEs7sTRn9h3czECyaunu1aFVw4sbbnqXDGGpoxX2lPCONEowDC4sLkN8Z57U7mLdnvJj8FFc4Vm7aWY4S10M0KpqIN7ekr4WTUvPAO/qal4rsQYYyYecF42LxJ4CJTBRVO7HevHjX/crGWSaQx7lwcuuueSUnBhfMyNPmZ8gA8r1gtqHDG97tt1RULxPa382DkQCdNXG6rYJHJMwQKJpxzFIVbwVxtu6aMNtrOi4x+69qd1GW85i2YcF4il18FCbE18IfWfLdHO9KHLb4/2m4YylaS665zzdDENl8+y4gV+COEsTcS/+jVPbOJLZhVkuRkV4iS/piFdkvnsxgB2dqxAMjTQ/OuXNTX1fUTn8mDr9jprY/cWiiGCqY5oTVt73NDc+7mgOm6HnvbBU8m3q6x2uXzbM1XEOFc3bR1HqZ8F9gFXVW1Z2N5ZdWRcGKO7KslJbt4jNd8BRHOgORk4Z0dfVK5Yz9vkL66Ow9jLBRz22kgdO2XxSyC7GQWeVxHoCDC6WSmjAE615Z4DT+MsmEtmgjI8S1LAaE9c8TMq+SeC2eo9f4Z2HVYYLeCsHM0CSP2ep117cxfu0V2cRmP+TwXTtgursF40/Yug0ZUk3AOqtH/Qdeu220caPHL44fr7JIQ+VxCwHPhdGTowchftjfc8b9GLJ5WNh6H/1fGsBzdMj9cl2MekdwDBDwVTn4EFQvntg+ZYaBp0poJfDDrdtS1U9G1J6D01dtT4ZwcrFiKGXKZbQTSTH6Y7mxShEHGsvh2qm3WRMb8I+CpcMokdrGB7VpE44vvyQTYwHvPGU8KJsdn82MAPCm2nZotoYj3FAFPhRM1czK2+9OO+po3UqHT9y//PACNvC9VnNUwycfHhq3WYbyl80w417S242AUrjax+7DU400DuZTjUUN8ZidjtrdTMxMWsXYR8Ew4ne5jYyKVUfiYpmeMzwYQlpRmhZq2fjxbOhHvHQKeCSe0pqOzQrgiJeOM/PC70Z9CgB3dgyTJstgt8k72spbkiXAub93CTxfaPu+Myc7vdiiRtzPVZu+WjacQ/7NMabLF4fCWEM5sIHkY74lwTiDFjnaF0OVa6rKxXmkpXVp8cbnB6pb7zksbLyI8RcAT4YSZhqMunTEtY5eeQEzXnZnQcTpBUuKI1wQv4u0cAdeF8/IN902g1NGuEKPv0X4rVf2jfvwFGILg5l/7D3ahRNduH7685nRdOCumF+OSKjrBNte4mqV7c807VvLvV5QorOmet5I2bRrKruJfsEgbLyI8Q8D1M0SSw7uJsHvzUXyKhE92rD34QI+1hKlTYTG/uGxKKT82/KPUKUSoVwi4LZzcNM7JrhCUbuybOyVeAcLLwQ0cvGsXwukl6CnKcrVbX9PUgQv4iemSqhQ8+C4IY+SVON3pKja+q7QPGXK1AWQnx38LChYtD0nTLy8oC6Lw+Hfm3ALijBVOQiRJErN2t+TCIl3XNOcKZev5WDz/mEU+fJcM3+MRwlngVnFtQlQsOzn+W2BUYsXTD/P7z/uUiP0rb/xQDYc8VLV2tmOKaDy9sLO7LrLRIVlL2V3TnPgS2Rm/04KPRVdaQnEcJ4JdA8wc6YcT/9ik8Gx71xXh5IvYWENacsa3meRs2/WMr3+BK+CKcJZOKbsaS0ierk26gSMW5D+9XLnPN7cMu1FHP9N0RTjJ8EdQ/Vxvq7zJE6SSlVYTnxXpYDTrVT3dEE4oHIe7Ql7V3kI52J9yfdYOwGATMPpgSGT6gsVoTEFcJl5g8+rIoDuXGuRdOCub2hcC7Bm5MOHntFAUy90+Nowy3jNigEmI59+7NJZvclNi4oVJ1MSrKW2ePflfSqLOjmPE6/c43pYskTLiwfRSrBrckjFNlkis1U6+hJ57Ja6y+3GWpPajKXkbmWcnCKDfnJxwF/pNGZsM+wYDG+yQweOqM+/CKTmd4TIyOHhYXZevz+uFWztWOtXkkhwzBHFPOAl9Ha08sl2KDznMdbXVcyCOS9fmmkRT01/LIbujpHnt1le2tF+I9bCPOuEIVxruyZdgxvjIfqQ4K7toHFfXbCnRk+56kv7eD/fVh5S2mai76eyXRKVfZgUsTwlimjPU1H4jBv4Xj9Kkr+Eu8P8Y9VtzFTk8YclLwXjL2TmgJFb5/Z3QnGuTgnPz4mu5/Dbm7Y21pu9A5kYkfWqmSc9S46cTYMlVOffvbujt6vpB+lzux9CAtD5usjhcGCOHuhsiv3O/5OESYpoTt13ciF9Ea+IfQ4zbbTLgWMNIeuYjwLnypQ5JeRH2gIvHhnsaq1+Kfa3CUDlJlu4JK1umGoI8dVY1d3wEhuJ3GgtlVP8vo99td0w4obpPGguC/xyj34p7sdIxERpqsZW06dJgIjBwQj/2i3TxdsLjn2V+w05eYx5g4viHZ6SX5MY33OgD5jDcjhIo7uI385nD3fctU+6tgMbsNW2k4A5UNqR/2/3SR0uICadOyFujQehaCbkEfrSH9WdqQF+OHPzDV/Yfxp7rVxTVPoF0OZ0PFfhtzPHz9+kKcRT+7hBrg/bks/aRBz3YZ84JVjzj5VeO+bVBZcGyvWjLpM9sk0d7ldrfjjDngWNYc1LyB2NZ0IBl4eatFxvDsrtl55olD5OXVHzix7Y7VXguYfil4pResfM6pim0X4mchHB+Abziz/QsQiP9Ntzc/g20CVcarjxcKMMtHd+VGX0BBSS1PTvwvnZ6gysFZyCaWEpKmi2iJSTpSuSztmyAIw2xow25KdsxbOks/gmXMTHOAlQSfRbn0Z0RQW6s+fHdogcdE0pDoLuhZju05F1YW73XlARfukPYJphAb4IAoZejvF1U/GDKIcn2KkbJSUrYcRylxtFt8iHQLE/ZfIwc11S6Kn6DtIkttz0xzdkzdPQVVNK0TYVBkOVtu8pA+adilXPCLUDo04+7MhveXr/xL9iFedUJe8N56dI1ilLqnE56Clgl+VddJ59Ld/4eQjoLwrSYd/kQpnlcsGz9EzIfbXY16H06bdvhMzoai17Rp1S/kp5j92JiwonDXBh2MtMiM7r2FTlc5O+4u8OSz/AnA12qKxrb+awdllayXLHMJRZHyPY2VD+kq2weOvguKA38efxAUaDQrw4e1ub3NdxpGvJ5ycmwcKJEdFnmNTU+uSmTIlaYARHnwun06uwsjGJZxLlw8jI8srjiFvjd9dXXR5k+mwsKND//aohpVSVLlXOKRhlv4v8HWGe+6YR65LyeuurNuW6GVDa1zc3n5kFizEleVY/tvDRYfhga02i0ccequzvbnvxmtanLN9Z6TXPbbCw7zDGG2XHnTXjSFK6q7++Wg6X4tkHKkVWaXGODgc/q2LHhWG8zNj7fIfHbnDeDLv+nGJOeT3Q2E93xFIwZizTCRhRMLmWjHirT6SmNqUdPnzr11jP33P3XXPKnSitJ8o3heUsWanM+uXab8uVjqdLkEjYinPwql8taOu5H03EQhh9KpgUnshZ40mtQymagy3Q0SQDIg27vPHCwcHPIZmiHcxPVs/teHpwy42lCDtnN7yAfw5iUL/vxf18+fCwMJfAi7iy4dltj5BdOmBwRTk5kUNU6iwOBL6GA0Qak9PZwc0dvT0NkV6qCttXX/gzh/N/3D7rJOt8zOQ4YRO90gSyT57DLtB5K5zt2q2TqEnYqtf+HLSosWYw+vBvE80is+x4NFi6BQEYEMGwoJhJ9EEtf37O7y2USTl5a79Cx72Pg/RNTyZRUyJL0lJs7JKbyhGfcIADF9rlzAuV7YCdwUa6VGiOcfFkJ3ftnsYBx3EgMv4TLSknx8/wstzFcuAUCWRHAjdE0UPxiZWvb8qxpDQnGCici0b0f0DR2LQQ0akjLnRfLQboPY4lrksKFVyCQGQFMriUm7Qi3tNcjIUaL2Z+Uwsmz9X0tslvT9Zvh1JLITMFY4glYmD8hxqFJyAhvZgRwnSX+msMtnX1WzAFNs/Vkyn2NNY9DS0YxJfohZN1kcYRu/hqZSiEU9Jiua9/pa1y/Jzm/8I9FIPT1jiW4SWQOVMevuusjzxtThFo7V0mMzQaee/oaa00W5+Gm9uuhFCqwOrmrW6kxbcVCUaxDLzfxtKr18F5vhCZsHsKB8mruR9zDfMKbiOMmjlMDZB33v6se/Xa/wRos1Hr/DIkFr0O7R7EYb1om5AoJ7b4KW4on+uojjyXo5fLGatBqFijeF1I6r8m0NZpROHmBWAr4Uaip7SoYv/4nBHKmkQkMdnn+m2VZvrmqteMA1jufwQ7DPnz9/FVC9WPyKelN45XZ/BgHrkWcbqQx3tzAaFqmOskyvQk/9M9jV3IL0j1vTCsxchswXS3L0l0INwknQRgEej5OP/4j4kzCia+ItIDmeUVy4PeIO5CgiZ2RAOi1cb9EAjvwGhHOSUSdRmkwFjeJkO8jbsRUkWqBi6hM29CWPL1JOGUi8/38NmnYVsGWcIIm5J5cIgXI3nBT5xdhbP0oD0t+AskBqfxcK8IA9eOlwdIHAH44VRoUh1vlyDqAsW74YkWMGCYyPr5oTaTHMQ4F+dcl/OJ9diMAeSklMnkEy02ffOvNlze++OCDQ0ZE0o45jYm4e5fylSM9dZEqwnQIJ+O/UPEIBPKCABRazawL5vXzA3VGgpaFM5Gpu76mt3vo6BxN09dC7Zu6pUQa8RYI5IoABPQKKSi9VNXUeWUir6VuPZF45I210D5CHof/cW6dzYgckihZinHPQnTv5SPpkh2UnsLe9kBy8HjyY1zIjX+D46lOntUFS5cqYe8nyrMnnIncePc01HKrbD645/+E38pWFCy6SNZxUwRhJgMFzPz4zDE2e+Rpx+OD8dNDsXH3eKycm3WCva2mnrrBaM3kWDiT+Y2b85t2l5LTnM1+TdW+phH9W4wExpiUaepQBEs0myQy9PYYjHT1hqjGJnDztuQ4WKsvxVGK4NDJ6OvGOFiaDV2kfGsuD3v74H5TvoMH9x/6wMw5sbgnFcX0nacjJ6KvTJsiz8XsP3mNm+gDA/+tlZbMxTn708aynLhh+X9v7yu7N5GuLlN5llbqnRR8tuUdozkZ+cbZYg0Fk8RWDO2+arnNYTytM3JLb0Mk5Xn4vGtOy4yJhGc1AlgT/wOuU6yCYKa9QSTn2fpZjaiofF4QwJGTntOqujCbgbnQnHmBWxCxhABuDdEJa+itr+GnLbBwk/kRwpkZHxGbJwSgLd+BON7U21CzyypJ0a1bRUqks48AYy+RoeiCdEd90hEWwpkOGRGeFwQw8fn3E+rRRT3Khj/nSlB067kiJtJbQwC7PRhfbsBp0U5rGcamEsI5FhMR4hABjC8PUp1d29tYs9cJKdGtO0FP5B2LAA5H6mToE90OBZMTFsI5Fl4RYhMBlepPYHy5tK/uzsM2SZiyiW7dBIdzD4ypf42j1U+NUsrH7Xaj1Pzs2lZX85Kf+RO8CQQEAgIBgYBAQCAgEBAICAQEAgIBgYBAQCAgEBAICAQEAgIBgYBAQCAgEBAI+B2B/wcrmpXY459pdgAAAABJRU5ErkJggg==\" alt=\"Active Countermeasures\" style=\"width:75px; float:left\" />"

var dbHeaderEmpty = `
<head>
<meta content="text/html;charset=utf-8" http-equiv="Content-Type">
<meta content="utf-8" http-equiv="encoding">
<link rel="stylesheet" type="text/css" href="../style.css">
</head>

<ul>
  <a href="http://activecountermeasures.com/" target="_blank">
	` + activecmImg + `
  </a>
  <li><a href="../index.html">RITA</a></li>
  <li><a href="index.html">Viewing: {{.DB}}</a></li>
</ul>
	`

var dbHeader = `
<head>
<meta content="text/html;charset=utf-8" http-equiv="Content-Type">
<meta content="utf-8" http-equiv="encoding">
<link rel="stylesheet" type="text/css" href="../style.css">
</head>

<ul>
  <a href="http://activecountermeasures.com/" target="_blank">
	` + activecmImg + `
  </a>
  <li><a href="../index.html">RITA</a></li>
  <li><a href="index.html">Viewing: {{.DB}}</a></li>
  <li><a href="beacons.html">Beacons</a></li>
	<li><a href="strobes.html">Strobes</a></li>
	<li><a href="dns.html">DNS</a></li>
  <li><a href="bl-source-ips.html">BL Source IPs</a></li>
	<li><a href="bl-dest-ips.html">BL Dest. IPs</a></li>
	<li><a href="bl-hostnames.html">BL Hostnames</a></li>
	<li><a href="long-conns.html">Long Connections</a></li>
	<li><a href="useragents.html">User Agents</a></li>
	<li style="float:right">
    <a href="https://github.com/activecm/rita" target="_blank">RITA on
		<img src="../github.svg" title="Icon made by Dave Gandy from www.flaticon.com" id="github">
    </a>
  </li>
</ul>
`

var homeHeader = `
<head>
<meta content="text/html;charset=utf-8" http-equiv="Content-Type">
<meta content="utf-8" http-equiv="encoding">
<link rel="stylesheet" type="text/css" href="./style.css">
</head>
<ul>
    <a href="http://activecountermeasures.com/" target="_blank">
		` + activecmImg + `
    </a>
  <li><a href="./index.html">RITA</a></li>
  <li style="float:right">
    <a href="https://github.com/activecm/rita" target="_blank">RITA on
		<img src="./github.svg" title="Icon made by Dave Gandy from www.flaticon.com" id="github">
    </a>
  </li>
</ul>
`

// Hometempl is our home template html
var Hometempl = homeHeader + `
<p>
  <div class="info">To view individual databases, click on any of the links below.</div>
  <div class="vertical-menu">
    {{range .}}
      <a href="{{.}}/index.html">{{.}}</a>
    {{end}}
  </div>
</p>
`

// DNStempl is our dns page template
var DNStempl = dbHeader + `
<div class="container">
  <table>
    <tr><th>Subdomain Count</th><th>Visited</th><th>Domain</th><tr>
    {{.Writer}}
  </table>
</div>
`

// DBhometempl is our database home template for each directory
var DBhometempl = dbHeader + `
<p>
  <div class="info">To view results, click on any of the links above.</div>
</p>
`

// DBemptyhometempl is our database home template for any empty database, or
// directory associated with a database in the MetaDatabase but doesn't exist
var DBemptyhometempl = dbHeaderEmpty + `
<p>
  <div class="info">{{.DB}} has no results. This might indicate that this database has been dropped, but the metadatabase hasn't been updated.
	<br>To update the metadatabase and fix this problem, run:
	<br>rita delete-database {{.DB}} </div>
</p>
`

// BeaconsTempl is our beacons html template
var BeaconsTempl = dbHeader + `
<div class="container">
  <table>
  <tr><th>Score</th><th>Source</th><th>Destination</th><th>Connections</th><th>Avg. Bytes</th><th>
	Intvl. Range</th><th>Size Range</th><th>Intvl. Mode</th><th>Size Mode</th><th>Intvl. Mode Count</th>
	<th>Size Mode Count</th><th>Intvl. Skew</th><th>Size Skew</th><th>Intvl. Dispersion</th><th>Size Dispersion
	</th></tr>
      {{.Writer}}
  </table>
</div>
`

//StrobesTempl is the strobes html template
var StrobesTempl = dbHeader + `
<div class="container">
  <table>
	<tr><th>Source</th><th>Destination</th><th>Connection Count</th></tr>
	  {{.Writer}}
	</table>
</div>
`

// BLSourceIPTempl is our blacklisted source ip html template
var BLSourceIPTempl = dbHeader + `
<div class="container">
  <table>
  <tr><th>IP</th><th>Connections</th><th>Unique Connections</th><th>Total Bytes</th><th>Destinations</th><tr>
    {{.Writer}}
  </table>
</div>
`

// BLDestIPTempl is our blacklisted destination ip html template
var BLDestIPTempl = dbHeader + `
<div class="container">
  <table>
  <tr><th>IP</th><th>Connections</th><th>Unique Connections</th><th>Total Bytes</th><th>Sources</th><tr>
    {{.Writer}}
  </table>
</div>
`

// BLHostnameTempl is our blacklisted hostname html template
var BLHostnameTempl = dbHeader + `
<div class="container">
  <table>
  <tr><th>Hostname</th><th>Connections</th><th>Unique Connections</th><th>Total Bytes</th><th>Sources</th><tr>
    {{.Writer}}
  </table>
</div>
`

// LongConnsTempl is our long connections html template
var LongConnsTempl = dbHeader + `
<div class="container">
  <table>
	<tr><th>Source</th><th>Destination</th><th>DstPort:Protocol:Service</th><th>Duration</th></tr>
	  {{.Writer}}
	</table>
</div>
`

// UserAgentsTempl is our user agents html template
var UserAgentsTempl = dbHeader + `
<div class="container">
  <table>
	<tr><th>User Agent</th><th>Times Used</th></tr>
	  {{.Writer}}
	</table>
</div>
`
