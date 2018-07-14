import React from 'react';
import { Button, Navbar, NavbarGroup, NavbarDivider } from "@blueprintjs/core";
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


export class Game extends React.Component {
  constructor(props) {
    super(props);
    console.log(props);
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
        <canvas
          id="threeCanvas"
          style={{ 
            backgroundColor:"rgb(0, 0, 0)",
            overflow:"hidden",
            top:"0",
            left:"0",
            width:"100%vh",
            height:"100vh"
          }}
        />
      </div>
    );
  }
}
