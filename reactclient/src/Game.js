import React, { Component } from 'react';
//import ReactDOM from 'react-dom';
//import * as Blueprint from '@blueprintjs/core';
import { Button, Navbar, NavbarGroup, NavbarDivider } from "../node_modules/@blueprintjs/core";
import { Systems } from "./menus/Systems.js";
import fscreen from "fscreen";

function requestFullscreen() {
  if (fscreen.fullscreenEnabled) {
    if (fscreen.fullscreenElement == null) {
      fscreen.requestFullscreen(document.body);
    } else {
      fscreen.exitFullscreen();
    }
  } else {
    window.alert("your browser doesnt support fullscreen")
  }
}
fscreen.addEventListener('fullscreenerror', function(e){console.log(e)}, {});

class Game extends Component {
  constructor(props) {
    super(props);
    this.state = {
        navbarOffset: null
    }
  }

  updateNavbarOffset = () => {
    var width = document.documentElement.clientWidth / 2
    this.setState({
      navbarOffset: width+"px"
    })
  }

  componentDidMount = () => {
    this.updateNavbarOffset();
    window.addEventListener("resize", this.updateNavbarOffset);
  }


  render() {
    return (
      <div 
        className="Game"
        style={{
          display:"flex",
          justifyContent:"center",
          //alignItems:"center",
        }}
      >
        <div
          style={{ 
            position:"fixed"
          }}
        >
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
