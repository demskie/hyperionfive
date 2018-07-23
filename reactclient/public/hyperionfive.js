const oneBillion = 1000000000

var newGameDataEvent = new Event("newgamedata");

var galaxy = new proto.gamestate.Galaxy();
var nameConvert = new Map();
var renderedPlanets = new Map();
var renderedObjects = new Map();
var renderedSystems = new Map();

var planets = [];
var timestamp = 0;
var scaleVector = new THREE.Vector3();

var socket;

var camera, renderer, controls;
var scene = new THREE.Scene();

var systemGeometry = new THREE.SphereGeometry(10000, 16, 12);
var greenMaterial = new THREE.MeshBasicMaterial({ color: 0x00ff00, wireframe: true });

if (!Detector.webgl) {
	window.alert("Sorry, your browser does not appear to support WebGL")
}



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
	console.debug("googleId:", googleId, "\naccessToken:", accessToken, "\nemail:", email, "\nname:", name,
		"\nimageUrl:", imageUrl, "\nisMobileOnly:", isMobileOnly, "\ndeviceDetectInfo:", deviceDetectInfo)


	watchSystems();

	createTestData();

	init();
	animate();

}



function handleWebsocketConnections() {
	socket = new WebSocket("wss://hyperion5.com/clienttofrontend");
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
}



function createTestData() {
	var systemAlpha = new proto.gamestate.System();
	systemAlpha.setInterx(0);
	systemAlpha.setIntery(0);
	galaxy.getSystemmapMap().set(0, systemAlpha);



	for (var i = 0; i < 10000; i++) {
		var sys = new proto.gamestate.System()
		sys.setNameuint(i + 1)

		let x = chance.integer({ min: -oneBillion, max: oneBillion })
		if (chance.bool()) { x *= -1 }

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
}



function init() {
	var canvasElement = document.getElementById("threejs");
	renderer = new THREE.WebGLRenderer({ antialias: true, canvas: canvasElement });

	camera = new THREE.PerspectiveCamera(50, canvasElement.clientWidth / canvasElement.clientHeight, 0.0001, 10 * oneBillion);
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
}





var planet;


function animate() {
	timestamp = Date.now() * 0.0001;

	requestAnimationFrame(animate);

	var speed = 4;
	var orbit = 3000000;
	planet.position.x = Math.cos(timestamp * speed) * orbit;
	planet.position.y = Math.sin(timestamp * speed) * orbit;

	//mesh.rotation.x += 0.01;

	controls.update();

	renderer.render(scene, camera);
}



var circleMap = new THREE.TextureLoader().load("textures/circle.svg");
var yellowCircleMaterial = new THREE.SpriteMaterial({ map: circleMap, color: "gold", fog: false });
var yellowCircleSprite = new THREE.Sprite(yellowCircleMaterial);
yellowCircleSprite.scale.set(1000000, 1000000, 1)

var redCircleMaterial = new THREE.SpriteMaterial({ map: circleMap, color: "red", fog: false });
var redCircleSprite = new THREE.Sprite(redCircleMaterial);
redCircleSprite.scale.set(500000, 500000, 1)





function watchSystems() {
	window.addEventListener("newgamedata", function () {
		for (var [k, v] of window.galaxy.getSystemmapMap().entries()) {
			let sys = renderedSystems.get(k)
			if (sys == undefined) {
				//sys = new THREE.Mesh(systemGeometry, greenMaterial)
				//var material = new THREE.SpriteMaterial({ map: circleMap, color: Math.random() * 0xffffff, fog: true });
				sys = yellowCircleSprite.clone();
				sys.position.x = v.getInterx();
				sys.position.y = v.getIntery();
				scene.add(sys);
				renderedSystems.set(k, sys)
			} else {
				sys.position.x = v.getInterx();
				sys.position.y = v.getIntery();
			}
		}

		planet = redCircleSprite.clone();
		scene.add(planet)
	})
}




// https://developer.mozilla.org/en-US/docs/Web/JavaScript/Reference/Global_Objects/Map
// https://github.com/google/protobuf/blob/master/js/map.js
// https://chancejs.com/location/city.html