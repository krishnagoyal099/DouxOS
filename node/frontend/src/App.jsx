import { useState, useEffect } from 'react';
import './App.css';

function App() {
    const [status, setStatus] = useState("Initializing...");
    const [logs, setLogs] = useState([]);

    useEffect(() => {
        // Listen for events from Go backend
        // Note: You need to set up Wails Events in app.go for this to work dynamically
        // For now, we just poll status
        const interval = setInterval(() => {
            try {
                window.go.main.App.GetStatus().then((res) => {
                    setStatus(res);
                });
            } catch (e) {
                console.log("Backend not ready");
            }
        }, 1000);

        return () => clearInterval(interval);
    }, []);

    return (
        <div className="container">
            <div className="header">
                <h1>DouxOS Node</h1>
                <div className="status-badge">
                    {status}
                </div>
            </div>

            <div className="content">
                <div className="node-visual">
                    {/* Cool placeholder for 3D grid or animation */}
                    <div className="pulse"></div>
                    <span>Running in Background</span>
                </div>
            </div>

            <div className="footer">
                <p>Contributing idle cycles to Science.</p>
            </div>
        </div>
    );
}

export default App;