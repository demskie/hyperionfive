import React, { Component } from 'react';
//import ReactDOM from 'react-dom';
//import * as Blueprint from '@blueprintjs/core';
import { Button, Navbar, NavbarGroup, NavbarDivider, Alignment } from "../node_modules/@blueprintjs/core";
import { Systems } from "./menus/Systems.js";

class Game extends Component {
  render() {
    return (
      <div className="Game">
        <div style={{ position:"fixed",
                      width:"425px",
                      left:"50%",
                      transform:"translateX(-50%)" }}>
          <Navbar className="bp3-dark">
            <NavbarGroup align={Alignment.LEFT}>
              <Systems />
              <Button className="bp3-minimal" icon="geolocation" text="Fleets" />
              <NavbarDivider />
              <Button className="bp3-minimal" icon="zoom-in" text="" />
              <Button className="bp3-minimal" icon="zoom-out" text="" />
              <NavbarDivider />
              <Button className="bp3-minimal" icon="property" text="" />
              <Button className="bp3-minimal" icon="cog" text="" />
              <NavbarDivider />
              <Button className="bp3-minimal" icon="fullscreen" text="" />
            </NavbarGroup>
          </Navbar>
        </div>
        <canvas id="threeCanvas"
          style={{ backgroundColor:"rgb(0, 0, 0)",
                   overflow:"hidden",
                   top:"0",
                   left:"0",
                   width:"100%vh",
                   height:"100vh" }}>
        </canvas>
      </div>
    );
  }
}

export default Game;
