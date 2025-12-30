import { BrowserRouter, Route, Routes } from "react-router-dom";
import EditorPage from "./pages/EditorPage.tsx";
import Home from "./pages/Home.tsx";
import KanbanPage from "./pages/KanbanPage.tsx";

function App() {
  return (
    <BrowserRouter>
      <Routes>
        <Route path="/" element={<Home />} />
        <Route path="/editor/:id" element={<EditorPage />} />
        <Route path="/kanban/:id" element={<KanbanPage />} />
      </Routes>
    </BrowserRouter>
  );
}

export default App;
