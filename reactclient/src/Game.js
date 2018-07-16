import React from 'react';
import { Button, Navbar, NavbarGroup, NavbarDivider, Alert } from "@blueprintjs/core";
import { Systems } from "./menus/Systems.js";
import fscreen from "fscreen";
import { Flex, Box } from 'reflexbox'



export class Game extends React.Component {
	constructor(props) {
		super(props);
		this.state = {
			fullscreenButtonDisabled: false,
			fullscreenAlertIsOpen: false
		}
	}

	requestFullscreen = () => {
		if (fscreen.fullscreenEnabled) {
			if (fscreen.fullscreenElement == null) {
				fscreen.requestFullscreen(document.body);
			} else {
				fscreen.exitFullscreen();
			}
		} else {
			this.setState({
				fullscreenAlertIsOpen: true
			})
		}
	}

	closeFullscreenAlert = () => {
		this.setState({
			fullscreenAlertIsOpen: false,
			fullscreenButtonDisabled: true,
		})
	}

	componentDidMount = () => {
		window.startGame(
			this.props.googleId,
			this.props.accessToken,
			this.props.email,
			this.props.name,
			this.props.imageUrl
		)
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
							<Button className="bp3-minimal" icon="fullscreen" text=""
								onClick={this.requestFullscreen}
								disabled={this.state.fullscreenButtonDisabled} />
						</NavbarGroup>
					</Navbar>
				</Box>
				<Box>
					<Alert
						className="bp3-minimal"
						confirmButtonText="Okay"
						isOpen={this.state.fullscreenAlertIsOpen}
						isClosed={this.closeFullscreenAlert}
					>
						<p>
							"Sorry, your browser does not appear to support fullscreen mode"
						</p>
					</Alert>
				</Box>
			</Flex >
		);
	}
}