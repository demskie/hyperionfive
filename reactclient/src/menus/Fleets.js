import React, { Component } from 'react';
import { Button, Overlay, Classes, H3, Code, Intent } from "../../node_modules/@blueprintjs/core";
import classNames from "classnames";

export class Fleets extends Component {
    constructor(props) {
        super(props);
        this.state = {
            visible: false
        }
    }

    openOverlay = () => {
        this.setState({
            visible: true
        });
    }

    closeOverlay = () => {
        this.setState({
            visible: false
        })
    }

    render() {
        return (
            <div>
                <Button 
                    className="bp3-minimal"
                    icon="search-around"
                    text="Fleets"
                    onClick={this.openOverlay} />
                <Overlay 
                    className={classNames(Classes.OVERLAY_SCROLL_CONTAINER, "bp3-dark")}
                    isOpen={this.state.visible}
                    onClose={this.closeOverlay}>
                    <div 
                        className={Classes.CARD}
                        id="theFleetsOverlay">
                        <H3>I'm an Overlay!</H3>
                        <p>
                            This is a simple container with some inline styles to position it on the screen. Its CSS
                            transitions are customized for this example only to demonstrate how easily custom
                            transitions can be implemented.
                        </p>
                        <p>
                            Click the right button below to transfer focus to the "Show overlay" trigger button outside
                            of this overlay. If persistent focus is enabled, focus will be constrained to the overlay.
                            Use the <Code>tab</Code> key to move to the next focusable element to illustrate this
                            effect.
                        </p>
                        <br />
                        <Button 
                            intent={Intent.DANGER} 
                            onClick={this.closeOverlay}>
                            Close
                        </Button>
                        <Button 
                            onClick={this.focusButton} 
                            style={{ float: "right" }}>
                            Focus button
                        </Button>
                    </div>
                </Overlay>
            </div>
        )
    }
}

