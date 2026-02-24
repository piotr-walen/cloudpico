import { useEffect, useState } from "react"
import { Link, Outlet } from "react-router"

const Layout = () => {
    const [qrModalOpen, setQrModalOpen] = useState(false)

    useEffect(() => {
        if (!qrModalOpen) return
        const onEscape = (e: KeyboardEvent) => {
            if (e.key === "Escape") setQrModalOpen(false)
        }
        document.addEventListener("keydown", onEscape)
        return () => document.removeEventListener("keydown", onEscape)
    }, [qrModalOpen])

    return (
        <div>
            <nav className="navbar">
                <Link to="/" className="navbar-brand">
                    <img src="/logo.png" alt="Logo" width={48} height={48} />
                    <span className="navbar-site-name">nubecita.casa</span>
                </Link>
                <button
                    type="button"
                    className="navbar-qr"
                    onClick={() => setQrModalOpen(true)}
                    aria-label="Show QR code to share"
                >
                    <img src="/qr.svg" alt="" width={40} height={40} />
                </button>
            </nav>
            <Outlet />
            {qrModalOpen && (
                <div
                    className="qr-modal-backdrop"
                    onClick={() => setQrModalOpen(false)}
                    role="dialog"
                    aria-modal="true"
                    aria-label="Share web panel"
                >
                    <div
                        className="qr-modal-content"
                        onClick={(e) => e.stopPropagation()}
                    >
                        <button
                            type="button"
                            className="qr-modal-close"
                            onClick={() => setQrModalOpen(false)}
                            aria-label="Close"
                        >
                            ×
                        </button>
                        <h2 className="qr-modal-title">Share web panel</h2>
                        <p className="qr-modal-hint">Scan to open on another device</p>
                        <a href="https://nubecita.casa" target="_blank" rel="noopener noreferrer" className="qr-modal-qr-link">
                            <img src="/qr.svg" alt="QR code for https://nubecita.casa" />
                        </a>
                        <p className="qr-modal-url">https://nubecita.casa</p>
                    </div>
                </div>
            )}
        </div>
    )
}

export default Layout