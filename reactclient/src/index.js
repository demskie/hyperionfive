import React from 'react';
import ReactDOM from 'react-dom';
import './index.css';
import { Game } from './Game';
import registerServiceWorker from './registerServiceWorker';
import { GoogleLogin } from "react-google-login";
import classNames from "classnames";


export class Main extends React.Component {
    constructor(props) {
        super(props);
        this.state = {
            googleId: null,
            accessToken: null,
            email: null,
            name: null,
            imageUrl: null,
        }
    }

    authSuccess = (response) => {
        if (this.state.googleId === null) {
            console.debug("authSuccess:", response)
            setTimeout(() => {
                // begin fading out the login page
                var loginPage = document.getElementById("loginPage")
                loginPage.className = "animated fadeOut"
                // fade the background color of the body to black
                document.body.style["-webkit-transition"] = "background-color 2000ms linear";
                document.body.style["-moz-transition"] = "background-color 2000ms linear";
                document.body.style["-ms-transition"] = "background-color 2000ms linear";
                document.body.style["-o-transition"] = "background-color 2000ms linear";
                document.body.style.transition = "background-color 2000ms linear";
                document.body.style.backgroundColor = "black"
                setTimeout(() => {
                    // move the entire loginPage offscreen for good measure
                    loginPage.style.position = "absolute"
                    loginPage.style.width = "1px"
                    loginPage.style.height = "1px"
                    loginPage.style.left = "-9999px"
                    loginPage.style.display = "none"
                    // pass the googleAuth information to googleAnalytics
                    window.ga('set', 'userId', response.googleId);
                    window.ga('create', 'UA-96532084-1', 'auto');
                    window.ga('send', 'pageview');
                    // update the component state
                    this.setState({
                        googleId: response.googleId,
                        accessToken: response.accessToken,
                        email: response.profileObj.email,
                        name: response.profileObj.name,
                        imageUrl: response.profileObj.imageUrl
                    })
                }, 2500);
            }, 1000);
        }
    }

    authFailure = (response) => {
        console.log("authFailure:", response)
    }

    componentDidMount = () => {
        // initialize google analytics
        if (typeof window.ga === "undefined") { 
            /*eslint-disable */
            (function(i,s,o,g,r,a,m){i['GoogleAnalyticsObject']=r;i[r]=i[r]||function(){
            (i[r].q=i[r].q||[]).push(arguments)},i[r].l=1*new Date();a=s.createElement(o),
            m=s.getElementsByTagName(o)[0];a.async=1;a.src=g;m.parentNode.insertBefore(a,m)
            })(window,document,'script','https://www.google-analytics.com/analytics.js','ga');
            /*eslint-enable */
        }
    }

    render() {
        if (this.state.googleId === null) {
            return (
                <div id="loginPage" className="animated fadeInUp" style={{display:"flex",flexDirection:"column",justifyContent:"center",alignItems:"center",width:"100%",height:"100vh"}}>
                    <div id="gameTitle" style={{color:"grey",marginBottom:"20px",fontSize:"70pt"}}>hyperion5</div>
                    <GoogleLogin
                        clientId="771328535291-3lv19jrlmhdd8f6hdhfd92qu3l7iugmd.apps.googleusercontent.com"
                        buttonText="Google Login"
                        onSuccess={this.authSuccess}
                        onFailure={this.authFailure}
                        className={classNames("bp3-button", "bp3-large", "bp3-intent-success")}
                        style={{marginBottom:"25vh"}}
                        isSignedIn={true}
                    />
                </div> 
            );
        } else {
            return (
                <Game 
                    googleId={this.state.googleId}
                    accessToken={this.state.accessToken}
                    email={this.state.email}
                    name={this.state.name}
                    imageUrl={this.state.imageUrl}
                />
            )
        }
    }
}

ReactDOM.render(<Main />, document.getElementById('root'));
registerServiceWorker();
