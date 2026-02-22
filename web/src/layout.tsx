import { Link, Outlet } from "react-router"

const Layout = () => {
    return (
        <div>
            <nav className="navbar">
                <Link to="/" className="navbar-brand">
                    <img src="/logo.png" alt="Logo" width={48} height={48} />
                    <span className="navbar-site-name">nubecita.casa</span>
                </Link>
            </nav>
            <Outlet />
        </div>
    )
}

export default Layout