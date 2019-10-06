var socket = new WebSocket("ws://" + window.location.host + "/api/measures");
var output = document.getElementById("data-field");
let params = [];
let values = [];
var LowerBound=0, UpperBound=0;
var selectParameter = 0;
var dataLength = 50;
var writeMessage = function(message) {
    //output.innerHTML += "<p>service: " + message + "</p>";
}

socket.onopen = function () {
    loadParametrs();
};

socket.onclose = function (event) {
    if (event.wasClean) {
        writeMessage('Connection closed gracefully');
    } else {
        writeMessage('Connection aborted');
    }

    writeMessage('code: ' + event.code + ' reason: ' + event.reason);
};

socket.onmessage = function(event) {
    var myJson = JSON.parse(event.data);
    var Time = new Date(myJson.Timestamp);
    Time.setMilliseconds(0);
    for(t in params)
    {
        for (var i in myJson.Parameters) 
        {
            if(i==params[t])
            {
                values[t][1].push({x: Time,y:myJson.Parameters[i]});
                if(values[t][1].length>dataLength)
                {
                    values[t][1].shift();
                }
            }
        }
    }
    Chart()
}


socket.onerror = function(error) {
    alert("error occurred: " + error.message);
}

function loadParametrs() {
    var myParams, i;
    var x = new XMLHttpRequest();
    x.open("GET", "http://" + window.location.host + "/api/parameters", true);
    x.onload = function ()
    {
        var j=0;
        myParams = JSON.parse(x.responseText);
        for (i in myParams) 
        {
            document.getElementById("parameters").innerHTML += "<option value="+ j++ +">" + myParams[i] + "</option>";
            params.push(myParams[i]);
            values.push([myParams[i],[]]);
        }
        loadBound(params[0]);
    }
    x.send(null);
    x.onerror = function()
    {
        alert("Не удалось загрузить параметры");
    }
}

function loadBound(myParams) {
    var myBound;
    var y = new XMLHttpRequest();
    y.open("GET", "http://" + window.location.host + "/api/" + myParams + "/bounds", true);
    y.onload = function ()
    {
        myBound=  JSON.parse(y.responseText);
        document.getElementById("UpperBound").value = myBound.UpperBound;
        document.getElementById("LowerBound").value = myBound.LowerBound;
        UpperBound = myBound.UpperBound;
        LowerBound = myBound.LowerBound;
        Chart();
    }
    y.send(null);
    y.onerror = function()
    {
        UpperBound = 0;
        LowerBound = 0;
        alert("Не удалось загрузить значения аварийных установок для: " + myParams);
    }
}

function selectedParameter() {
    selectParameter = document.getElementById("parameters").value;
    loadBound(params[selectParameter]);
    Chart();
}

function editBound() {
    var d = document.getElementById("parameters").value;
    var a = document.getElementById("UpperBound").value;
    var b = document.getElementById("LowerBound").value;
    if (a.indexOf(',') > -1)
    {
        a = a.split(',').join('.');
        document.getElementById("UpperBound").value = a; 
    }
    if (b.indexOf(',') > -1)
    {
        b = b.split(',').join('.');
        document.getElementById("LowerBound").value = b;
    }
    setBound(params[d],a,b);
}

function setBound(myParams, a, b) {
    var z = new XMLHttpRequest();
    z.open("PATCH", "http://" + window.location.host + "/api/" + myParams + "/bounds", true);
    z.setRequestHeader("Content-Type", "application/x-www-form-urlencoded");
    z.send('{"LowerBound": ' + b + ',"UpperBound": ' + a + '}');
    z.onerror = function()
    {
        alert("Не удалось изменить значения аварийных установок для: " + myParams);
    }
    z.onreadystatechange = function() { 
        if (this.status === 200) {
            loadBound(myParams);
        }
    }
}

function Chart() {
    var chart = new CanvasJS.Chart("chartContainer", {
	exportEnabled: true,
	title :{
		text: values[selectParameter][0]
	},
	axisY: {
		includeZero: false,
		stripLines: [{
			value:UpperBound,
            label: "",
            labelFontColor: "#FF0000"//цвет для верхней границы
		}]
    },
    axisY2: {
        includeZero: false,
		stripLines: [{
			value:LowerBound,
            label: "",
            labelFontColor: "#FF0000"//цвет для нижней границы
		}]
    },
    axisX:{     
        valueFormatString: "hh:mm:ss",
      },
	data: [{
        name: "Type 1 Filter",
        type: "line",
        color: "#000FFF",//цвет для линий должен быть одинаковым
		markerSize: 0,
		dataPoints: values[selectParameter][1] 
    },
    {
        name: "",
        type: "line",
        markerSize: 0,
        color: "#000FFF",//цвет для линий должен быть одинаковым
        showInLegend: false,
        axisYType: "secondary",
		dataPoints: values[selectParameter][1] 
    }]
    });
    chart.render();
}