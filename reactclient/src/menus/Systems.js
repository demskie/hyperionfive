import React, { Component } from 'react';
import { Button, Dialog, Classes } from "@blueprintjs/core";
//import { Table, Column } from "@blueprintjs/table";
import classNames from "classnames";
import { Flex, Box } from 'reflexbox'



export class Systems extends Component {
	constructor(props) {
		super(props);
		this.state = {
			visible: false,
			dialogMaxBodyHeight: null,
			dialogWindowWidth: "85%"
		}
	}

	openDialog = () => {
		this.setState({
			visible: true
		});
	}

	closeDialog = () => {
		this.setState({
			visible: false
		})
	}

	updateDimensions = () => {
		var height = document.documentElement.clientHeight - 260
		if (height < 200) {
			height = 200
		}
		var width = document.documentElement.clientWidth * 0.85
		if (width > 2000 * 0.85) {
			width = 2000 * 0.85
		}
		this.setState({
			dialogMaxBodyHeight: height + "px",
			dialogWindowWidth: width + "px"
		})
	}

	componentDidMount = () => {
		window.addEventListener("resize", this.updateDimensions);
		window.addEventListener("deviceorientation", this.updateDimensions);
		this.updateDimensions();
	}

	detectClickAndHold = (val) => {
		console.log("detected click against", val)
		window.alert("detected a click")
	}
	// zooming out to galaxy view shows
	//	1) battle lines
	// 	2) 



	// system view
	// filter: production rate
	// filter: system pressure
	// filter: proximity
	// search: system name
	// clicking: makes the webgl canvas zoom to that system
	// holding: makes the 


	// idea: spin rate represents production rate
	// and the color of spinner represents threat level

	render() {
		return (
			<div>
				<Button
					className="bp3-minimal"
					icon="search-around"
					text="Systems"
					onClick={this.openDialog}
				/>
				<Dialog
					className={classNames("bp3-dark")}
					//icon="info-sign"
					title="Systems"
					isOpen={this.state.visible}
					onClose={this.closeDialog}
					style={{
						width: this.state.dialogWindowWidth,
						height: "100%",
					}}
				>
					<div
						id="systemScrollableDialogBody"
						style={{
							overflow: "hidden",
							overflowY: "scroll",
							maxHeight: this.state.dialogMaxBodyHeight,
							marginBottom: "15px"
						}}
					>
						<div className={Classes.DIALOG_BODY}>
							<Flex wrap>
								<Box>
									<div className="bp3-card bp3-elevation-3 bp3-interactive"
										style={{ width: "300px", height: "140px", marginLeft: "10px", marginRight: "10px", marginTop: "10px", marginBottom: "10px" }}
										onClick={() => this.detectClickAndHold("Andromeda")}>
										<p> Andromeda </p>
										<p> Size: Large </p>
										<p> Threat: Significant </p>
										<p> Status: Under Siege </p>
									</div>
								</Box>
								<Box>
									<div className="bp3-card bp3-elevation-3 bp3-interactive"
										style={{ width: "300px", height: "140px", marginLeft: "10px", marginRight: "10px", marginTop: "10px", marginBottom: "10px" }}
										onClick={() => this.detectClickAndHold("Andromeda")}>
										<p> Andromeda </p>
										<p> Size: Large </p>
										<p> Threat: Significant </p>
										<p> Status: Under Siege </p>
									</div>
								</Box>
								<Box>
									<div className="bp3-card bp3-elevation-3 bp3-interactive"
										style={{ width: "300px", height: "140px", marginLeft: "10px", marginRight: "10px", marginTop: "10px", marginBottom: "10px" }}
										onClick={() => this.detectClickAndHold("Andromeda")}>
										<p> Andromeda </p>
										<p> Size: Large </p>
										<p> Threat: Significant </p>
										<p> Status: Under Siege </p>
									</div>
								</Box>
								<Box>
									<div className="bp3-card bp3-elevation-3 bp3-interactive"
										style={{ width: "300px", height: "140px", marginLeft: "10px", marginRight: "10px", marginTop: "10px", marginBottom: "10px" }}
										onClick={() => this.detectClickAndHold("Andromeda")}>
										<p> Andromeda </p>
										<p> Size: Large </p>
										<p> Threat: Significant </p>
										<p> Status: Under Siege </p>
									</div>
								</Box>
								<Box>
									<div className="bp3-card bp3-elevation-3 bp3-interactive"
										style={{ width: "300px", height: "140px", marginLeft: "10px", marginRight: "10px", marginTop: "10px", marginBottom: "10px" }}
										onClick={() => this.detectClickAndHold("Andromeda")}>
										<p> Andromeda </p>
										<p> Size: Large </p>
										<p> Threat: Significant </p>
										<p> Status: Under Siege </p>
									</div>
								</Box>
								<Box>
									<div className="bp3-card bp3-elevation-3 bp3-interactive"
										style={{ width: "300px", height: "140px", marginLeft: "10px", marginRight: "10px", marginTop: "10px", marginBottom: "10px" }}
										onClick={() => this.detectClickAndHold("Andromeda")}>
										<p> Andromeda </p>
										<p> Size: Large </p>
										<p> Threat: Significant </p>
										<p> Status: Under Siege </p>
									</div>
								</Box>
								<Box>
									<div className="bp3-card bp3-elevation-3 bp3-interactive"
										style={{ width: "300px", height: "140px", marginLeft: "10px", marginRight: "10px", marginTop: "10px", marginBottom: "10px" }}
										onClick={() => this.detectClickAndHold("Andromeda")}>
										<p> Andromeda </p>
										<p> Size: Large </p>
										<p> Threat: Significant </p>
										<p> Status: Under Siege </p>
									</div>
								</Box>
								<Box>
									<div className="bp3-card bp3-elevation-3 bp3-interactive"
										style={{ width: "300px", height: "140px", marginLeft: "10px", marginRight: "10px", marginTop: "10px", marginBottom: "10px" }}
										onClick={() => this.detectClickAndHold("Andromeda")}>
										<p> Andromeda </p>
										<p> Size: Large </p>
										<p> Threat: Significant </p>
										<p> Status: Under Siege </p>
									</div>
								</Box>
								<Box>
									<div className="bp3-card bp3-elevation-3 bp3-interactive"
										style={{ width: "300px", height: "140px", marginLeft: "10px", marginRight: "10px", marginTop: "10px", marginBottom: "10px" }}
										onClick={() => this.detectClickAndHold("Andromeda")}>
										<p> Andromeda </p>
										<p> Size: Large </p>
										<p> Threat: Significant </p>
										<p> Status: Under Siege </p>
									</div>
								</Box>
								<Box>
									<div className="bp3-card bp3-elevation-3 bp3-interactive"
										style={{ width: "300px", height: "140px", marginLeft: "10px", marginRight: "10px", marginTop: "10px", marginBottom: "10px" }}
										onClick={() => this.detectClickAndHold("Andromeda")}>
										<p> Andromeda </p>
										<p> Size: Large </p>
										<p> Threat: Significant </p>
										<p> Status: Under Siege </p>
									</div>
								</Box>
							</Flex>
						</div>
					</div>
					<div className={Classes.DIALOG_FOOTER}>
						<div className={Classes.DIALOG_FOOTER_ACTIONS}>
							{/*
							<Tooltip content="This button is hooked up to close the dialog.">
								<Button onClick={this.closeDialog}>Close</Button>
							</Tooltip>
							<AnchorButton
								intent={Intent.PRIMARY}
								href="https://www.palantir.com/palantir-foundry/"
								target="_blank"
							>
								Visit the Foundry website
							</AnchorButton>
							*/}
						</div>
					</div>
				</Dialog>
			</div>
		)
	}
}

