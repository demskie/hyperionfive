function startGame(login_type, id_token, id) {

	var Galaxy = new proto.main.Galaxy();

	// baseline checks
	if (typeof(Worker) == "undefined"){
		alert("Sorry, your browser does not support WebWorkers. Halting.");
		throw("webworkers unsupported by this browser");
	}
	if (!window.WebSocket){
		alert("Sorry, your browser does not support WebSockets. Halting.");
		throw("websockets unsupported by this browser");
	}


	// send data to loadbalance.hyperion5.com
	function contactLoadBalancer() {
		var socket = new WebSocket("wss://hyperion5.com/loadbalance");
		socket.onopen = function(event) {
			console.log("sending id_token to wss://hyperion5.com/loadbalance");
			socket.send(JSON.stringify(
				{
					"login_type" : login_type,
					"id_token" : id_token,
				}
			));
			socket.binaryType = 'arraybuffer';
		}
		socket.onmessage = function(event) {
			//var incomingMsg = JSON.parse(event.data);
			//console.log("received the following message:");
			//console.log(JSON.stringify(incomingMsg));

			var myGalaxy = new proto.main.Galaxy();

			//Galaxy = proto.main.Galaxy.deserializeBinary(Array.from(event.data));

			var someData = proto.main.Galaxy.deserializeBinary(event.data)

			console.log(someData);
			console.log(someData.getName());
			someData.setName("Andromeda-999");
			console.log(someData.getSolarsystemsliceList());



			//var arrayOfGalaxies = new Array();
			//for(objIter in incomingMsg["Servers"]) {
			//	arrayOfGalaxies.push( incomingMsg["Servers"][objIter]["GalaxyName"] )
			//}
			//console.log("  arrayOfGalaxies: " + arrayOfGalaxies);

			//pingTestGameServers(incomingMsg["Servers"]);
			//return incomingMsg["Servers"];

				//	"Servers": [{
				//	    "GalaxyName": "Milkyway-832",
				//	    "ServerList": ["127.0.0.1", "blahblahblah"]
				//	}, {
				//	    "GalaxyName": "Andromeda-187",
				//	    "ServerList": ["10.0.0.99", "192.168.1.1", "moredata"]
				//	}]
				//	}
		}
		socket.onclose = function(event) {
			console.log("connection closed!!!");
			console.log(event);
		}
		socket.onerror = function(event) {
			console.log("error detected!!!");
			console.log(event);
		}
	}
	var validLoadBalanceResponse = contactLoadBalancer();

	function pingTestGameServers(data) {
		var socket = new WebSocket("ws://echo.websocket.org");
		socket.onopen = function(event) {
			console.log("pingTest - sending message to ws://echo.websocket.org")
			socket.send("hello there");
		}
		socket.onmessage = function(event) {
			console.log("pingTest - received message from ws://echo.websocket.org");
			console.log(event.data);
		}
		socket.onclose = function(event) {
			console.log("pingTest - connection closed!!!");
			console.log(event);
		}
		socket.onerror = function(event) {
			console.log("pingTest - error detected!!!");
			console.log(event);
		}
	}



	//var wsJson = new WebSocket("ws://hyperion5.com/socketserver", "hyperionJson");
	//var wsProtobuf = new WebSocket("ws://hyperion5.com/socketserver", "hyperionProtobuf");


	// accessing data structures defined by protobuf.js
	var Galaxy = new proto.main.Galaxy();
	//var Planet = new proto.main.Planet();
	//var SolarSystem = new proto.main.SolarSystem();
	//var Spaceship = new proto.main.Spaceship();

	function testData() {
		Galaxy.setName("Andromeda-187");
		Galaxy.setSolarsystemsliceList([
			"something",
			"somethingelse"
		]);
		console.log(Galaxy);
	}
	//testData();

	//////////////////////////////////////////////////
	//////////////////////////////////////////////////
	//////////////////////////////////////////////////

	if ( ! Detector.webgl ) Detector.addGetWebGLMessage();

	var camera, controls, scene, renderer;

	init();
	//render(); // remove when using next line for animation loop (requestAnimationFrame)
	animate();

	//////////////////////////////////////////////////

	function init() {

		scene = new THREE.Scene();


		//////////////////////////////////////////////////

		renderer = new THREE.WebGLRenderer();
		//renderer.setClearColor( scene.fog.color );
		renderer.setPixelRatio( window.devicePixelRatio );
		renderer.setSize( window.innerWidth, window.innerHeight );

		//////////////////////////////////////////////////

		document.body.appendChild( renderer.domElement );
		//var container = document.getElementById( 'container' );
		//container.appendChild( renderer.domElement );

		//////////////////////////////////////////////////

		camera = new THREE.PerspectiveCamera( 45, window.innerWidth / window.innerHeight, 0.1, 1000 );
		camera.position.z = 10;

		//////////////////////////////////////////////////

		controls = new THREE.OrbitControls( camera, renderer.domElement );
		//controls.addEventListener( 'change', render ); // remove when using animation loop
		controls.target.set( 0, 0, 0 ); // view direction perpendicular to XY-plane

		//////////////////////////////////////////////////

		var geometry = new THREE.BoxGeometry( 1, 1, 1 );
		//var material = new THREE.MeshBasicMaterial( { color: 0x00ff00 } );
		//var cube = new THREE.Mesh( geometry, material );
		var wireframe = new THREE.WireframeGeometry( geometry );
		var line = new THREE.LineSegments( wireframe );
		line.material.depthTest = true;
		line.material.opacity = 0.25;
		line.material.transparent = false;

		scene.add( line );

		//////////////////////////////////////////////////

		window.addEventListener( 'resize', onWindowResize, false );

	}

	//////////////////////////////////////////////////

		//var render = function () {
		//	requestAnimationFrame( render );

		//	//controls.update();

		//	line.rotation.x += 0.001;
		//	line.rotation.y += 0.001;

		//	renderer.render( scene, camera );

		//	//onWindowResizeCanvas();
		//};
		//render();

	function onWindowResize() {
		camera.aspect = window.innerWidth / window.innerHeight;
		camera.updateProjectionMatrix();
		renderer.setSize( window.innerWidth, window.innerHeight );
	}

	function animate() {
		requestAnimationFrame( animate );
		controls.update(); // required if controls.enableDamping = true, or if controls.autoRotate = true
		//stats.update();
		render();
	}

	function render() {
		renderer.render( scene, camera );
	}

}

//////////////////////////////////////////////////

// now that everything has finished loading, hide the "downloading..." text
document.getElementById("loadingStatus").style.visibility = "hidden";

// set initial scale for googleLoginButton
var loginButtonScaleBy = (document.documentElement.clientWidth / 120) / 3
document.getElementById("googleLoginButton").style.transform = "translate(-50%, 0px) scale("+loginButtonScaleBy+")";

//////////////////////////////////////////////////

function onWindowResize() {
	// update googleLoginButton
	if (document.getElementById("googleLoginButton").style.visibility !== "hidden") {
		var loginButtonScaleBy = (document.documentElement.clientWidth / 120) / 3
		document.getElementById("googleLoginButton").style.transform = "translate(-50%, 0px) scale("+loginButtonScaleBy+")";
	}
}
window.addEventListener( 'resize', onWindowResize, false );
