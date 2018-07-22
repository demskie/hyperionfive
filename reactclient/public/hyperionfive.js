var newGameDataEvent = new Event("newgamedata");
var galaxy = new proto.gamestate.Galaxy();
var nameConvert = new Map();

function debounce(func, timeout) {
	var timer;
	return function (event) {
		if (timer) clearTimeout(timer);
		timer = setTimeout(func, timeout, event);
	};
}

var distanceFunction = function (ax, ay, az, bx, by, bz) {
	return Math.pow(ax - bx, 2) + Math.pow(ay - by, 2) + Math.pow(az - bz, 2);
};

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
	socket.onmessage = function (ev) {
	};

	socket.onclose = function (ev) {
		console.log("connection closed!!!", ev);
	};
	socket.onerror = function (ev) {
		console.log("error detected!!!", ev);
	}

	// accessing data structures defined by protobuf.js

	var systemAlpha = new proto.gamestate.System();
	systemAlpha.setInterx(0);
	systemAlpha.setIntery(0);
	galaxy.getSystemmapMap().set(0, systemAlpha);

	var oneBillion = 1000000000
	for (var i = 0; i < 10000; i++) {
		var sys = new proto.gamestate.System()
		sys.setNameuint(i + 1)
		if (chance.bool()) {
			sys.setInterx(Math.random() * oneBillion)
		} else {
			sys.setInterx(Math.random() * -oneBillion)
		}
		if (chance.bool()) {
			sys.setIntery(Math.random() * oneBillion)
		} else {
			sys.setIntery(Math.random() * -oneBillion)
		}
		galaxy.getSystemmapMap().set(i + 1, sys);
	}

	for (var i = 0; i < 10001; i++) {
		nameConvert.set(i, chance.city())
	}

	window.dispatchEvent(newGameDataEvent);



	// ensure that this device supports webgl
	if (!Detector.webgl) {
		window.alert("Sorry, your browser does not appear to support WebGL")
	}

	var camera, scene, renderer;
	var geometry, material, mesh;
	var controls;

	init();
	animate();

	function init() {
		var canvasElement = document.getElementById("threejs");
		renderer = new THREE.WebGLRenderer({ antialias: false, canvas: canvasElement });

		camera = new THREE.PerspectiveCamera(50, canvasElement.clientWidth / canvasElement.clientHeight, 0.0001, 10000000);
		camera.position.z = 1000000;

		controls = new THREE.MapControls(camera, renderer.domElement);
		controls.dampingFactor = 0.025;
		controls.screenSpacePanning = true;
		controls.enableZoom = true;
		controls.enableRotate = false;
		controls.enableKeys = false;

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
		geometry = new THREE.BoxGeometry(100000, 100000, 100000); // 100,000
		material = new THREE.MeshNormalMaterial();

		mesh = new THREE.Mesh(geometry, material);
		scene.add(mesh);
	}

	function animate() {
		requestAnimationFrame(animate);

		mesh.rotation.x += 0.01;
		mesh.rotation.y += 0.02;

		controls.update();

		renderer.render(scene, camera);
	}
}

// https://developer.mozilla.org/en-US/docs/Web/JavaScript/Reference/Global_Objects/Map
// https://github.com/google/protobuf/blob/master/js/map.js
// https://chancejs.com/location/city.html
