
var username = "NOBODY"
var powerLinked = true; 
var eventws = new WebSocket("ws://" +window.location.host + "/events");
var userset = false;
var dir = 0; 

document.onkeydown = checkKey;

window.onload =  function () { 

username = getCookie("username"); 
if( username.length > 0 ) {
    document.getElementById('coverDiv').className += ' hiddenClass';    
    document.getElementById('usernameDiv').className += ' hiddenClass';
	userset = true;
}

};  
 
eventws.onmessage = function(event) {
    
    var msg = JSON.parse(event.data);

    ev = JSON.parse(msg.Event) 

    switch(msg.Type) {
        
        // ACTION 
        case 32: 
            handleEvent(ev, "actionLog") 
            break;
        case 64: 
            handleEvent(ev, "chatLog") 
            break;
        default: 
            console.log("Unknown event: ", msg.Type) 
    }
}

function checkKey(e) {

    e = e || window.event;

	switch(e.keyCode) {
		case 38:
        	e.preventDefault();
        	if(dir === -1 ) {
            	fullStop(); 
        	} else { 
            	fullForward();
        	}
			break;  
		case 40: 
        	e.preventDefault();
        	if( dir === 1 ) { 
            	fullStop();
        	} else { 
            	fullReverse(); 
        	}
			break; 
		case 37: 
        	e.preventDefault();
        	rotateLeft();
			break;
		case 39: 
        	e.preventDefault();
        	rotateRight();
			break;
		case 13: 
			sendChat(); 
			break; 
		default: 
	}


}

function handleEvent(ev, id) {
   var elem = document.getElementById(id);
   children = elem.children;
    
   if( children.length > 100 ) {
        elem.removeChild(elem.firstChild);    
   }

   var node = document.createElement("div");
   var textnode = document.createTextNode(ev.Time + ": " + ev.Name  + " > " + ev.Action);
   
   node.appendChild(textnode); 
   elem.appendChild(node); 
   elem.scrollTop = elem.scrollHeight;
}

function setUser() {
    username = document.getElementById("nameInput").value;
    document.getElementById('coverDiv').className += ' hiddenClass';    
    document.getElementById('usernameDiv').className += ' hiddenClass';
	userset = true;
	setCookie("username", username, 1); 
    return false; 
}

function load() {

}

function post( address, message ) {
    var method = "POST";
    var xhr = new XMLHttpRequest();
    xhr.open(method, address, true);
    xhr.setRequestHeader("Content-Type", "application/json");
    xhr.send(message);
}

function updatePower() {
    pl = document.getElementById("powerLeft");
    pr = document.getElementById("powerRight");
    
    var info = {};
    info["Name"] = username 
    info["Left"] = Number(pl.value) - 255;
    info["Right"] = Number(pr.value) - 255;
    console.log("Power: " + JSON.stringify(info)); 
    post("/power", JSON.stringify(info))
}

function updatePowerLinked(e) {

    if(powerLinked)  {
        if(e.id == "powerLeft") {
             pr = document.getElementById("powerRight");
             pr.value = e.value; 
        } else if (e.id == "powerRight") {
            pl = document.getElementById("powerLeft");
            pl.value = e.value; 
        }
    }

    updatePower();
}

function powerRight(p) {
    pr = document.getElementById("powerRight");
    pr.value = p; 
}

function powerLeft(p) {
    pl = document.getElementById("powerLeft");
    pl.value = p;   
}

function rotateLeft() {
    powerRight(255 * 2);
    powerLeft(0); 
    updatePower();
}

function rotateRight() {
    powerRight(0);
    powerLeft(255 * 2); 
    updatePower();
}

function fullForward() {
    powerRight(510);
    powerLeft(510); 
    updatePower();
}

function fullStop() {
    powerRight(255); 
    powerLeft(255); 
    updatePower();
}

function fullReverse() {
    powerRight(0); 
    powerLeft(0); 
    updatePower();
}

function toggleLinked() {

    b = document.getElementById("linkedButton"); 
 
    if(powerLinked) {
        powerLinked = false;
        b.innerHTML = "&nhArr;" 
    } else {
        powerLinked = true; 
        b.innerHTML = "&hArr;" 
    }
}


function sendChat() {
	
	if(!userset) {
		return;
	}

	var info = {};
	info["Name"] = username;
	info["Text"] = document.getElementById("txtArea").value;
	post("/chat", JSON.stringify(info));
	document.getElementById("txtArea").value = '';
}

function onTextChange() {
    var key = window.event.keyCode;

    if (key == 13) {
        //document.getElementById("txtArea").value =document.getElementById("txtArea").value + "\n*";
    
        var info = {};
        info["Name"] = username;
        info["Text"] = document.getElementById("txtArea").value;
        post("/chat", JSON.stringify(info));
        
        document.getElementById("txtArea").value = '';
            
        return false;
    } else {
        return true;
    }   
}

function setCookie(cname, cvalue, exdays) {
    var d = new Date();
    d.setTime(d.getTime() + (exdays*24*60*60*1000));
    var expires = "expires="+d.toUTCString();
    document.cookie = cname + "=" + cvalue + "; " + expires;
}

function getCookie(cname) {
    var name = cname + "=";
    var ca = document.cookie.split(';');
    for(var i=0; i<ca.length; i++) {
        var c = ca[i];
        while (c.charAt(0)==' ') c = c.substring(1);
        if (c.indexOf(name) == 0) return c.substring(name.length,c.length);
    }
    return "";
}



