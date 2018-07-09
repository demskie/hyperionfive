function goFullscreen() {
	
}


function startGame(login_type, id_token, id) {

	var firstToolbar = { 
		id:"firstToolbar",
		view:"toolbar", 
		elements:[
			{ view:"button", value:"Add", width:100 },
			{ view:"button", id:"fullscreen", value:"Fullscreen", width:120}
		]
	}

	var secondToolbar = {
		id:"secondToolbar",
		view:"toolbar",
		elements:[
			{ view:"button", value:"Show Galaxy", width:120 },
		]
	}

	webix.ui.fullScreen();
	var gui = webix.ui({
		type:"clean",
		margin:0,
		padding:0, 
		rows: [ 
			firstToolbar,
			{ 
				view:"template", template:"<canvas id='threeCanvas'></canvas>", type:"clean"
			},
			secondToolbar,
		]
	});
	console.log(gui)

	// hide the login buttons
	document.getElementById("loginPage").style.visibility = "hidden"

	// toggle fullscreen using the button
	gui.attachEvent("onClick", function(id){
		console.log("requesting fullscreen")
		if (id == "fullscreen") {
			document.body.requestFullscreen();
			document.body.webkitRequestFullScreen();
		}
	})

	// send data to hyperion5.com/clienttofrontend
	var socket = new WebSocket("wss://hyperion5.com/clienttofrontend");
	socket.onopen = function(event) {
		console.log("sending id_token to wss://hyperion5.com/clienttofrontend");
		socket.send(JSON.stringify({
			"login_type" : login_type,
			"id_token" : id_token
		}));
		socket.binaryType = 'arraybuffer';
	};
	socket.onmessage = function(event) {
		incomingMsg = JSON.parse(event.data);
		//console.log("received the following message:");
		//console.log(JSON.stringify(incomingMsg));

		var Galaxy = new proto.main.Galaxy();

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
	
		socket.onclose = function(event) {
			console.log("connection closed!!!", event);
		};
		socket.onerror = function(event) {
			console.log("error detected!!!", event);
		}
	}

	// send data to hyperion5.com/clienttobackend
	//	var socket = new WebSocket("wss://hyperion5.com/clienttobackend");
	//	socket.onopen = function(event) {
	//		console.log("sending id_token to wss://hyperion5.com/clienttofrontend");
	//		socket.send(JSON.stringify({
	//			"login_type" : login_type,
	//			"id_token" : id_token
	//		}));
	//		socket.binaryType = 'arraybuffer';
	//	};

	//var wsJson = new WebSocket("ws://hyperion5.com/socketserver", "hyperionJson");
	//var wsProtobuf = new WebSocket("ws://hyperion5.com/socketserver", "hyperionProtobuf");


	// accessing data structures defined by protobuf.js
	var Galaxy = new proto.gamestate.Galaxy();
	function testData() {
		console.log(Galaxy);

		var systemAlpha = new proto.gamestate.System();
		var planetAlpha = new proto.gamestate.Planet();
		planetAlpha.setInterx(-25);
		planetAlpha.setIntery(-25);
		var planetBravo = new proto.gamestate.Planet();
		planetBravo.setInterx(25);
		planetBravo.setIntery(-25);
		var planetCharlie = new proto.gamestate.Planet();
		planetCharlie.setInterx(25);
		planetCharlie.setIntery(25);
		var planetDelta = new proto.gamestate.Planet();
		planetDelta.setInterx(-25);
		planetDelta.setIntery(25);

		Galaxy.getSystemmapMap().set(3, systemAlpha);
		Galaxy.getPlanetmapMap().set(4, planetAlpha);
		Galaxy.getPlanetmapMap().set(5, planetBravo);
		Galaxy.getPlanetmapMap().set(6, planetCharlie);
		Galaxy.getPlanetmapMap().set(7, planetDelta);
		
		console.log(Galaxy);
	}
	testData();

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

		var canvasElement = document.getElementById("threeCanvas");
		renderer = new THREE.WebGLRenderer({canvas: canvasElement});
		//renderer.setClearColor( scene.fog.color );
		renderer.setPixelRatio( window.devicePixelRatio );
		renderer.setSize( canvasElement.clientWidth, canvasElement.clientHeight, false );
		window.addEventListener( 'resize', function(){
			renderer.setPixelRatio( window.devicePixelRatio );
			renderer.setSize( canvasElement.clientWidth, canvasElement.clientHeight, false );
		}, false);

		//////////////////////////////////////////////////

		camera = new THREE.PerspectiveCamera( 45, canvasElement.clientWidth / canvasElement.clientHeight, 0.1, 1000 );
		camera.position.z = 10;
		window.addEventListener( 'resize', function(){
			camera.aspect = canvasElement.clientWidth / canvasElement.clientHeight;
			camera.updateProjectionMatrix();
		}, false);

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