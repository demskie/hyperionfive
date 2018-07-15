import React, { Component } from 'react';
import { Button, Dialog, Classes, Tooltip, AnchorButton, Intent } from "@blueprintjs/core";
import classNames from "classnames";

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


	render() {
		return (
			<div>
				<Button className="bp3-minimal"
					icon="search-around"
					text="Systems"
					onClick={this.openDialog}
				/>
				<Dialog className={classNames("bp3-dark")}
					//icon="info-sign"
					title="Systems"
					isOpen={this.state.visible}
					onClose={this.closeDialog}
					style={{
						width: this.state.dialogWindowWidth,
						height: "100%",
					}}
				>
					<div id="systemScrollableDialogBody"
						style={{
							overflow: "hidden",
							overflowY: "scroll",
							maxHeight: this.state.dialogMaxBodyHeight,
							marginBottom: "15px"
						}}
					>
						<div className={Classes.DIALOG_BODY}>
							<p>
								<strong>
									Data integration is the seminal problem of the digital age. For over ten years, we’ve
									helped the world’s premier organizations rise to the challenge.
								</strong>
							</p>
							<p>
								Palantir Foundry radically reimagines the way enterprises interact with data by amplifying
								and extending the power of data integration. With Foundry, anyone can source, fuse, and
								transform data into any shape they desire. Business analysts become data engineers — and
								leaders in their organization’s data revolution.
							</p>
							<p>
								Foundry’s back end includes a suite of best-in-class data integration capabilities: data
								provenance, git-style versioning semantics, granular access controls, branching,
								transformation authoring, and more. But these powers are not limited to the back-end IT
								shop.
							</p>
							<p>
								In Foundry, tables, applications, reports, presentations, and spreadsheets operate as data
								integrations in their own right. Access controls, transformation logic, and data quality
								flow from original data source to intermediate analysis to presentation in real time. Every
								end product created in Foundry becomes a new data source that other users can build upon.
								And the enterprise data foundation goes where the business drives it.
							</p>
							<p>Start the revolution. Unleash the power of data integration with Palantir Foundry.</p>
							<p>
								<strong>
									Data integration is the seminal problem of the digital age. For over ten years, we’ve
									helped the world’s premier organizations rise to the challenge.
								</strong>
							</p>
							<p>
								Palantir Foundry radically reimagines the way enterprises interact with data by amplifying
								and extending the power of data integration. With Foundry, anyone can source, fuse, and
								transform data into any shape they desire. Business analysts become data engineers — and
								leaders in their organization’s data revolution.
							</p>
							<p>
								Foundry’s back end includes a suite of best-in-class data integration capabilities: data
								provenance, git-style versioning semantics, granular access controls, branching,
								transformation authoring, and more. But these powers are not limited to the back-end IT
								shop.
							</p>
							<p>
								In Foundry, tables, applications, reports, presentations, and spreadsheets operate as data
								integrations in their own right. Access controls, transformation logic, and data quality
								flow from original data source to intermediate analysis to presentation in real time. Every
								end product created in Foundry becomes a new data source that other users can build upon.
								And the enterprise data foundation goes where the business drives it.
							</p>
							<p>Start the revolution. Unleash the power of data integration with Palantir Foundry.</p><p>
								<strong>
									Data integration is the seminal problem of the digital age. For over ten years, we’ve
									helped the world’s premier organizations rise to the challenge.
								</strong>
							</p>
							<p>
								Palantir Foundry radically reimagines the way enterprises interact with data by amplifying
								and extending the power of data integration. With Foundry, anyone can source, fuse, and
								transform data into any shape they desire. Business analysts become data engineers — and
								leaders in their organization’s data revolution.
							</p>
							<p>
								Foundry’s back end includes a suite of best-in-class data integration capabilities: data
								provenance, git-style versioning semantics, granular access controls, branching,
								transformation authoring, and more. But these powers are not limited to the back-end IT
								shop.
							</p>
							<p>
								In Foundry, tables, applications, reports, presentations, and spreadsheets operate as data
								integrations in their own right. Access controls, transformation logic, and data quality
								flow from original data source to intermediate analysis to presentation in real time. Every
								end product created in Foundry becomes a new data source that other users can build upon.
								And the enterprise data foundation goes where the business drives it.
							</p>
							<p>Start the revolution. Unleash the power of data integration with Palantir Foundry.</p>
						</div>
					</div>
					<div className={Classes.DIALOG_FOOTER}>
						<div className={Classes.DIALOG_FOOTER_ACTIONS}>
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
						</div>
					</div>
				</Dialog>
			</div>
		)
	}
}

