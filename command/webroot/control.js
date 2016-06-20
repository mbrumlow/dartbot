
var ws; 
var powerLinked = true; 
var username = ""; 
var authenticated = false;
var dir = 0; 

document.onload = createWebSocket(); 

function createWebSocket() {

	ws = new WebSocket("ws://" +window.location.host + "/client");  

	ws.onopen =  function () { 

		// Attempt to log in using cookie information. 

		clearChatLog();

		username = getCookie("username"); 
		var authToken = getCookie("authToken");
		authToken = authToken.replace(/['"]+/g, '')

		if( username.length > 0  && authToken.length > 0 ) {

			ws.send(JSON.stringify({
				Name: username,
				Token: authToken,
			}));

			autoAuth(); 
		}

	};  

	ws.onmessage = function(event) {

		var msg = JSON.parse(event.data);

		ev = JSON.parse(msg.Event) 

		switch(msg.Type) {

			case 1: // AUTH_OK  
				authOk(msg.Event); 
				break;

			case 2: // AUTH_USERNAME_IN_USE
				authUserInUse();
				break; 

			case 3: // AUTH_REQUIRE_PASSWORD 
				authPassRequired();
				break;

			case 4: // AUTH_BAD_PASSWORD
				authBadPass(); 
				break;

			case 5: // AUTH_BAD_NAME
				authBadName(); 
				break;

				// ACTIONS

			case 32: // TrackPower
				handleEvent(msg, ev, "actionLog") 
				break;
			case 64: // Chat 
				handleEvent(msg, ev, "chatLog") 
				break;
			default: 
				console.log("Unknown event: ", msg.Type) 
		}

	};

	ws.onclose = function() {
		setTimeout(function() {
			createWebSocket()
		}, 3);
	};  
}

function autoAuth() {
    document.getElementById('authScreen').className = 'hidden';    
    document.getElementById('authInput').className = 'hidden';
    document.getElementById('authErrorInUse').className = 'authError hidden';
    document.getElementById('authErrorPassRequired').className = 'authError hidden';
    document.getElementById('AuthErrorBadPass').className = 'authError hidden';
}

function authOk(token) {

    document.getElementById('authScreen').className = 'hidden';    
    document.getElementById('authInput').className = 'hidden';
    document.getElementById('authErrorInUse').className = 'authError hidden';
    document.getElementById('authErrorPassRequired').className = 'authError hidden';
    document.getElementById('AuthErrorBadPass').className = 'authError hidden';

    setCookie("authToken", token) 
    setCookie("username", username) 

    authenticated = true;
}

function authUserInUse() {

    document.getElementById('authScreen').className = 'visible';    
    document.getElementById('authInput').className = 'visible';
    document.getElementById('authErrorInUse').className = 'authError visible';
    document.getElementById('authErrorPassRequired').className = 'authError hidden';
    document.getElementById('AuthErrorBadPass').className = 'authError hidden';
    document.getElementById('AuthErrorBadName').className = 'authError hidden';
       
    authenticated = false;
}

function authPassRequired() {

    document.getElementById('authScreen').className = 'visible';    
    document.getElementById('authInput').className = 'visible';
    document.getElementById('authErrorInUse').className = 'authError hidden';
    document.getElementById('authErrorPassRequired').className = 'authError visible';
    document.getElementById('AuthErrorBadPass').className = 'authError hidden';
    document.getElementById('AuthErrorBadName').className = 'authError hidden';
       
    authenticated = false;
}

function authBadPass() {

    document.getElementById('authScreen').className = 'visible';    
    document.getElementById('authInput').className = 'visible';
    document.getElementById('authErrorInUse').className = 'authError hidden';
    document.getElementById('authErrorPassRequired').className = 'authError hidden';
    document.getElementById('AuthErrorBadPass').className = 'authError visible';
    document.getElementById('AuthErrorBadName').className = 'authError hidden';
       
    authenticated = false;
}

function authBadName() {

    document.getElementById('authScreen').className = 'visible';    
    document.getElementById('authInput').className = 'visible';
    document.getElementById('authErrorInUse').className = 'authError hidden';
    document.getElementById('authErrorPassRequired').className = 'authError hidden';
    document.getElementById('AuthErrorBadPass').className = 'authError hidden';
    document.getElementById('AuthErrorBadName').className = 'authError visible';
       
    authenticated = false;
}

var chatEnabled = false; 
function chatSelected(selected) {
	console.log("Chat enabled", selected); 
	chatEnabled = selected
}

document.onkeydown = function checkKey(e) {

	e = e || window.event;

	switch(e.keyCode) {
		case 38:
			if(!chatEnabled) {
				e.preventDefault();
				if(dir === -1 ) {
					fullStop(); 
				} else { 
					fullForward();
				}
			}
			break;  
		case 40: 
			if(!chatEnabled) { 
				e.preventDefault();
				if( dir === 1 ) { 
					fullStop();
				} else { 
					fullReverse(); 
				}
			}
			break; 
		case 37:
			if(!chatEnabled) { 
				e.preventDefault();
				if( dir == 2 ) {
					fullStop();
				} else { 
					rotateLeft();
				}
			}
			break;
		case 39:
			if(!chatEnabled) {  
				e.preventDefault();
				if( dir == 3 ) { 
					fullStop();
				} else { 
					rotateRight();
				}
			}
			break;
		case 13:
			if(chatEnabled) {  
				sendChat(); 
			}
			break; 
		default: 
	}
}

function clearChatLog() {
	var node = document.getElementById("chatLog");
	while (node.firstChild) {
    	node.removeChild(node.firstChild);
	}
}

function handleEvent(je, ev, type) {

	var elem = document.getElementById(type);
	children = elem.children;

	if( children.length > 100 ) {
		elem.removeChild(elem.firstChild);    
	}

	var lastChatIndex = -1;
	if( children.length > 0 ) {
		lastChatIndex = elem.lastChild.getAttribute("chatIndex");
	}

	if( lastChatIndex < ev.Id ) {
		insertEventFast(je, ev, type) 	
	} else {
		insertEventSlow(je, ev, type) 	
	}

} 

function insertEventSlow(je, ev, type) {
	
	var elem = document.getElementById(type);
	children = elem.children;

	var node = document.createElement("div");
	var textnode = document.createTextNode(ev.Time + ": " + je.UserInfo.Name  + " > " + ev.Action);
	node.setAttribute("chatIndex", ev.Id); 
	node.appendChild(textnode); 

	for(var i = 0; i < children.length; i++) {
		lastChatIndex = children[i].getAttribute("chatIndex");
		if(lastChatIndex > ev.Id) {
			elem.insertBefore(node, children[i]);
			elem.scrollTop = elem.scrollHeight;	
			return 
		}	
	}
	
	elem.appendChild(node); 
	elem.scrollTop = elem.scrollHeight;

}

function insertEventFast(je, ev, type) {
	
	var elem = document.getElementById(type);

	var node = document.createElement("div");
	var textnode = document.createTextNode(ev.Time + ": " + je.UserInfo.Name  + " > " + ev.Action);
	node.setAttribute("chatIndex", ev.Id); 
	node.appendChild(textnode); 
	
	elem.appendChild(node); 
	elem.scrollTop = elem.scrollHeight;

}

function login() {

    username = document.getElementById("nameInput").value; 
    password = document.getElementById("passInput").value; 

    ws.send(JSON.stringify({
        Name: username,
        Auth: password,
    }));

    return false;

}

/* Not needed any more - YAY!
function post( address, message ) {
    var method = "POST";
    var xhr = new XMLHttpRequest();
    xhr.open(method, address, true);
    xhr.setRequestHeader("Content-Type", "application/json");
    xhr.send(message);
}
*/

function updatePower() {
    pl = document.getElementById("powerLeft");
    pr = document.getElementById("powerRight");
    
    var power = {};
    power["Left"] = Number(pl.value) - 255;
    power["Right"] = Number(pr.value) - 255;
   
    var je = {}; 
    je["Type"] = 2; // TrackPower
    je["Event"] = JSON.stringify(power);
    
    ws.send(JSON.stringify(je));
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
    dir = 3; 
    powerRight(255 * 2);
    powerLeft(0); 
    updatePower();
}

function rotateRight() {
    dir = 2; 
    powerRight(0);
    powerLeft(255 * 2); 
    updatePower();
}

function fullForward() {
    dir = 1; 
    powerRight(510);
    powerLeft(510); 
    updatePower();
}

function fullStop() {
    dir = 0; 
    powerRight(255); 
    powerLeft(255); 
    updatePower();
}

function fullReverse() {
    dir = -1;
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
    
    if(!authenticated) {
        return;
    }

	if( document.getElementById("txtArea").value.length == 0 ) {
		return;
	}

	var info = {};
	info["Type"] = 64; // CHAT_EVENT
	info["Event"] = document.getElementById("txtArea").value;

	

    ws.send(JSON.stringify(info));
    document.getElementById("txtArea").value = '';
    
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


