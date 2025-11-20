import { createRouter, createWebHistory } from "vue-router";
import Home from "@/pages/Home.vue";
import Dashboard from "@/pages/Dashboard.vue";

const routes = [
  {
    path: "/",
    name: "Home",
    component: Home,
  },
  {
    path: "/dashboard/:symbol?",
    name: "Dashboard",
    component: Dashboard,
    props: true,
  },
];

const router = createRouter({
  history: createWebHistory(import.meta.env.BASE_URL),
  routes,
});

export default router;
