function OnNewGameData(state) {
	var evt = new CustomEvent("authsuccess", { detail: state });
	window.dispatchEvent(evt);
}

function debounce(func, timeout) {
	var timer;
	return function (event) {
		if (timer) clearTimeout(timer);
		timer = setTimeout(func, timeout, event);
	};
}

function startGame(googleId, accessToken, email, name, imageUrl, isMobileOnly, deviceDetectInfo) {
	console.debug("googleId:", googleId)
	console.debug("accessToken:", accessToken)
	console.debug("email:", email)
	console.debug("name:", name)
	console.debug("imageUrl:", imageUrl)
	console.debug("isMobileOnly:", isMobileOnly)
	console.debug("deviceDetectInfo:", deviceDetectInfo)

	// send data to hyperion5.com/clienttofrontend
	var socket = new WebSocket("wss://hyperion5.com/clienttofrontend");
	socket.onopen = function (ev) {
		console.log("sending id_token to wss://hyperion5.com/clienttofrontend");
		socket.send(JSON.stringify({
			"login_type": "google",
			"id_token": accessToken
		}));
		socket.binaryType = 'arraybuffer';
	};
	socket.onmessage = function (event) {
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

		socket.onclose = function (event) {
			console.log("connection closed!!!", event);
		};
		socket.onerror = function (event) {
			console.log("error detected!!!", event);
		}
	}

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

	// ensure that this device supports webgl
	if (!Detector.webgl) {
		Detector.addGetWebGLMessage();
	}

	var camera, scene, renderer;
	var geometry, material, mesh;

	init();
	animate();

	function init() {
		var canvasElement = document.getElementById("threejs");
		renderer = new THREE.WebGLRenderer({ antialias: false, canvas: canvasElement });

		camera = new THREE.PerspectiveCamera(70, canvasElement.clientWidth / canvasElement.clientHeight, 0.01, 10);
		camera.position.z = 1;

		window.addEventListener("resize", window.debounce(function () {
			canvasElement.style.width = document.documentElement.clientWidth + "px"
			canvasElement.style.height = document.documentElement.clientHeight + "px"
			canvasElement.clientWidth = document.documentElement.clientWidth
			canvasElement.clientHeight = document.documentElement.clientHeight
			canvasElement.width = document.documentElement.clientWidth
			canvasElement.height = document.documentElement.clientHeight
			renderer.setSize(canvasElement.width, canvasElement.height);
			renderer.setPixelRatio(window.devicePixelRatio);
			camera.aspect = canvasElement.width / canvasElement.height;
			camera.updateProjectionMatrix();
		}, 100));
		renderer.setSize(canvasElement.clientWidth, canvasElement.clientHeight);
		renderer.setPixelRatio(window.devicePixelRatio);

		scene = new THREE.Scene();

		geometry = new THREE.BoxGeometry(0.2, 0.2, 0.2);
		material = new THREE.MeshNormalMaterial();

		mesh = new THREE.Mesh(geometry, material);
		scene.add(mesh);
		canvasElement.addEventListener('resize', function () {
			camera.lookAt(mesh.position);
		});
	}

	function animate() {
		requestAnimationFrame(animate);
		mesh.rotation.x += 0.01;
		mesh.rotation.y += 0.02;
		renderer.render(scene, camera);
	}

}