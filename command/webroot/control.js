
var powerLinked = true; 

function load() {
    fullStop();   
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
    
    console.log("POWER: L:", pl.value, " R:", pr.value);
    
    poutl = document.getElementById("leftPower"); 
    poutr = document.getElementById("rightPower"); 
   
    poutl.innerHTML = pl.value; 
    poutr.innerHTML = pr.value; 

    var info = {};
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
    updatePower()
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

