import React, { Component } from 'react';
import { Button, Dialog, Classes } from "@blueprintjs/core";
//import { Table, Column } from "@blueprintjs/table";
import classNames from "classnames";
import { Flex, Box } from 'reflexbox';
import { ModifiedSpinner } from "../elements/ModifiedSpinner";
import { default as OverflowEllipsis } from 'react-overflow-ellipsis'
import { Collection, Grid } from 'react-virtualized';
import "./Systems.css";


export class Systems extends Component {
	constructor(props) {
		super(props);
		this.state = {
			visible: false,
			actualSystemDialogBodyWidth: 0,
			actualSystemDialogBodyHeight: 0,
			systemGrid: [[]],
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
		var width = document.documentElement.clientWidth * 0.85
		if (width > 2000 * 0.85) {
			width = 2000 * 0.85
		}
		var height = document.documentElement.clientHeight - 260
		if (height < 200) {
			height = 200
		}
		this.setState({
			actualSystemDialogBodyWidth: width,
			actualSystemDialogBodyHeight: height,
		})
		this.resizeSystemGrid()
	}

	makeSystemGrid = () => {
		var newGrid = [[]];
		var cellWidth = 280
		var dialogWidth = document.documentElement.clientWidth * 0.85
		if (dialogWidth > 2000 * 0.85) {
			dialogWidth = 2000 * 0.85
		}
		var cellsPerRow = Math.floor((dialogWidth - 40) / cellWidth);
		var currentRow = 0;
		var currentColumn = 0;
		var receiveClick = (name) => {
			this.detectClickAndHold(name)
		}
		for (var [k, v] of window.galaxy.getSystemmapMap().entries()) {
			newGrid[currentRow].push(
				<div className="bp3-card bp3-elevation-2 bp3-interactive"
					style={{ width: "260px", height: "120px", marginLeft: "10px", marginRight: "10px", marginTop: "10px", marginBottom: "10px" }}
					onClick={
						function (name) {
							return function () {
								receiveClick(name)
							}
						}(k)
					}
				>
					<Flex align="center" style={{ height: "100%" }}>
						<Box style={{ width: "150px" }}>
							<OverflowEllipsis>
								<b>{window.nameConvert.get(k)}</b> <br />
								Size: Large <br />
								Threat: Significant<br />
								Status: Under Siege
							</OverflowEllipsis>
						</Box>
						<Box style={{ width: "60px", height: "60px" }}>
							<ModifiedSpinner intent="PRIMARY" size="60" cycleTime={window.chance.integer({ min: 1, max: 15 })} />
						</Box>
					</Flex>
				</div>
			);
			currentColumn++
			if (currentColumn >= cellsPerRow) {
				currentColumn = 0;
				currentRow++
				newGrid.push([]);
			}
		}
		this.setState({
			systemGrid: newGrid
		})
	}

	resizeSystemGrid = () => {
		var newGrid = [[]];
		var cellsPerRow = Math.floor((this.state.actualSystemDialogBodyWidth - 40) / 280);
		var currentRow = 0;
		var currentColumn = 0;
		for (var i = 0; i < this.state.systemGrid.length; i++) {
			for (var j = 0; j < this.state.systemGrid[i].length; j++) {
				newGrid[currentRow].push(this.state.systemGrid[i][j])
				currentColumn++
				if (currentColumn >= cellsPerRow) {
					currentColumn = 0;
					currentRow++
					newGrid.push([])
				}
			}
		}
		this.setState({
			systemGrid: newGrid
		})
	}

	cellRenderer = ({ columnIndex, key, rowIndex, style }) => {
		return (
			<div
				key={key}
				style={style}
			>
				{this.state.systemGrid[rowIndex][columnIndex]}
			</div>
		)
	}

	componentDidMount = () => {
		window.addEventListener("resize", this.updateDimensions);
		this.updateDimensions();

		window.addEventListener("newgamedata", () => {
			console.log(window.galaxy)
			this.updateDimensions()
			this.makeSystemGrid()
		});
	}

	detectClickAndHold = (val) => {
		console.log("detected click against", val)
		//window.alert("detected a click")
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
			<div style={{ zIndex: 9999 }}>
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
						width: this.state.actualSystemDialogBodyWidth + "px",
						height: "100%",
					}}
				>
					<div id="actualSystemDialogBody" className={Classes.DIALOG_BODY}>
						<Grid
							cellRenderer={this.cellRenderer}
							columnCount={this.state.systemGrid[0].length}
							columnWidth={280}
							height={this.state.actualSystemDialogBodyHeight}
							rowCount={this.state.systemGrid.length}
							rowHeight={140}
							width={this.state.actualSystemDialogBodyWidth - 40}
						/>
					</div>
				</Dialog>
			</div>
		)
	}
}

