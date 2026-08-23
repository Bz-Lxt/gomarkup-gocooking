import type { ReactNode } from "react";
import { Navigate, Route, Routes } from "react-router-dom";
import { getToken } from "./api";
import Layout from "./pages/Layout";
import Login from "./pages/Login";
import Calendar from "./pages/Calendar";
import Shopping from "./pages/Shopping";
import Recipes from "./pages/Recipes";
import Pantry from "./pages/Pantry";
import Settings from "./pages/Settings";

function Guard({ children }: { children: ReactNode }) {
  if (!getToken()) return <Navigate to="/login" replace />;
  return <>{children}</>;
}

export default function App() {
  return (
    <Routes>
      <Route path="/login" element={<Login />} />
      <Route
        path="/"
        element={
          <Guard>
            <Layout />
          </Guard>
        }
      >
        <Route index element={<Calendar />} />
        <Route path="shopping" element={<Shopping />} />
        <Route path="recipes" element={<Recipes />} />
        <Route path="pantry" element={<Pantry />} />
        <Route path="settings" element={<Settings />} />
      </Route>
      <Route path="*" element={<Navigate to="/" replace />} />
    </Routes>
  );
}
