<!DOCTYPE html>
<html lang="en">
<head>
    <title>Beltane - the Mayday Dump</title>
    <link href='//fonts.googleapis.com/css?family=Roboto:400,700,300' rel='stylesheet' type='text/css'>
    <link href='//fonts.googleapis.com/css?family=Roboto+Mono' rel='stylesheet' type='text/css'>
<style>
body {
    font-family: 'Roboto', sans-serif;
}

h1 {
    font-weight: 700;
}

p {
    font-weight: 300;
}

#container {
  background-color: #EEEEEE;
  width: 700px;
}

.code {
  font-family: 'Roboto Mono', monospace;
}
</style>
<script>
var get = function(name) {
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
  if (data.pruned) { document.getElementById("download").style.display = "none"; }
  else { document.getElementById("download").style.display = "block"; }
  document.getElementById("download").href = "http://localhost:8080/raw/" + data.sha + ".tar.gz";

  document.getElementById("container").style.display = "block";
}
</script>
<body>
<h1>Beltane - the Mayday Dump</h1>
<div style="display: none;" id="container">
<h2>sha1: <span id="sha"></span></h2>
<h3><a href="" id="download">download</a></h3>
<h3>Machine ID:</h3>
<div id="machine_id"></div>
</div>

<h2>All Dumps</h2>
<ul>

</ul>


<div style="font-size: 0.8em;">100% final design</div>
</body>
</html>
