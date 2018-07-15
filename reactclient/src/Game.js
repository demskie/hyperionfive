import React from 'react';
import { Button, Navbar, NavbarGroup, NavbarDivider } from "@blueprintjs/core";
import { Systems } from "./menus/Systems.js";
import fscreen from "fscreen";
import { Flex, Box } from 'reflexbox'

function requestFullscreen() {
	if (fscreen.fullscreenEnabled) {
		if (fscreen.fullscreenElement == null) {
			fscreen.requestFullscreen(document.body);
		} else {
			fscreen.exitFullscreen();
		}
	} else {
		window.alert("Sorry, your browser does not appear to support fullscreen mode")
	}
}

export class Game extends React.Component {
	constructor(props) {
		super(props);
		console.log(props);
		this.state = {
			deviceWidth: document.documentElement.clientWidth + "px",
			deviceHeight: document.documentElement.clientHeight + "px"
		}
	}



	componentDidMount = () => {

	}

	render() {
		return (
			<Flex align="center" column>
				<Box>
					<Navbar className="bp3-dark">
						<NavbarGroup>
							<Systems />
							<Button className="bp3-minimal" icon="geolocation" text="Fleets" />
							<NavbarDivider />
							<Button className="bp3-minimal" icon="zoom-in" text="" />
							<Button className="bp3-minimal" icon="zoom-out" text="" />
							<NavbarDivider />
							<Button className="bp3-minimal" icon="property" text="" />
							<Button className="bp3-minimal" icon="cog" text="" />
							<NavbarDivider />
							<Button className="bp3-minimal" icon="fullscreen" text="" onClick={requestFullscreen} />
						</NavbarGroup>
					</Navbar>
				</Box>
			</Flex>
		);
	}
}