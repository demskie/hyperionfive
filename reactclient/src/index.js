import React from 'react';
import ReactDOM from 'react-dom';
import './index.css';
import Game from './Game';
import registerServiceWorker from './registerServiceWorker';

function startReacting() {
    ReactDOM.render(<Game />, document.getElementById('root'));
    registerServiceWorker();
}

if (typeof ID_TOKEN === "undefined") {
    document.getElementById("loadingStatus").style.visibility = "hidden";
    document.getElementById("googleLoginButton").style.visibility = "visible";
    window.addEventListener("authsuccess", function(e){
        document.getElementById("loginPage").style.visibility = "hidden";
        document.getElementById("googleLoginButton").style.visibility = "hidden";
        startReacting()
    })
} else {
    document.getElementById("loginPage").style.visibility = "hidden";
    document.getElementById("googleLoginButton").style.visibility = "hidden";
    startReacting()
}


