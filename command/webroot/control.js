
var username = "NOBODY"
var powerLinked = true; 
var eventws = new WebSocket("ws://" +window.location.host + "/events");

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
    info["Left"] = Number(pl.value);
    info["Right"] = Number(pr.value);
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
    powerRight(180);
    powerLeft(0); 
    updatePower();
}

function rotateRight() {
    powerRight(0);
    powerLeft(180); 
    updatePower();
}

function fullForward() {
    powerRight(180);
    powerLeft(180); 
    updatePower();
}

function fullStop() {
    powerRight(90); 
    powerLeft(90); 
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



