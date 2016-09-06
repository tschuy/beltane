<html>
<head>
    <title>Mayday Dump listing</title>
</head>
<script>
var get = function(e, name) {
  console.log(e);
  console.log(name);

  var request = new XMLHttpRequest();
  request.open('GET', '/dump/' + name, true);

  request.onload = function() {
    if (request.status >= 200 && request.status < 400) {
      // Success!
      var data = JSON.parse(request.responseText);
      display(data);
    } else {
      // We reached our target server, but it returned an error
    }
  };

  request.onerror = function() {
    // There was a connection error of some sort
  };

  request.send();
}

var display = function(data) {
  console.log(data);
  document.getElementById("machine_id").innerHTML = data.machine_id;
  document.getElementById("sha").innerHTML = data.sha;
  document.getElementById("download").href = "http://localhost:8080/raw/" + data.sha + ".tar.gz";

  document.getElementById("container").style.display = "block";
}
</script>
<body>
<h1>Mayday</h1>
<ul>
{{ range $name := . }}
    <li><a onclick="get(this, '{{ $name }}'); return false;" href="#">{{ $name }}</a></li>
{{ end }}
</ul>

<div style="display: none;" id="container">
<h3>Machine ID:</h3>
<div id="machine_id"></div>
<h3>sha1 hash:</h3>
<div id="sha"></div>
<h3><a href="" id="download">download</a></h3>
</body>
</html>
