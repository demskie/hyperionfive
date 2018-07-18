import React, { Component } from 'react';
import { Spinner } from "@blueprintjs/core";
import "./ModifiedSpinner.css";

export class ModifiedSpinner extends Component {
	render() {
		return (
			<div style={{
				width: this.props.size + "px",
				WebkitAnimation: "spin360 " + this.props.cycleTime + "s linear infinite",
				animation: "spin360 " + this.props.cycleTime + "s linear infinite"
			}}>
				<Spinner
					// "NONE", "PRIMARY", "SUCCESS", "WARNING", "DANGER"
					intent={this.props.intent}
					size={this.props.size}
					value="0.25"
				/>
			</div>
		)
	}
}
